#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
usage: scripts/analysis-env.sh [--root DIR] [--ref REF] [--source-root DIR]

Create an isolated analysis prefix with the current source versions of the
jwa91 CLI tools built and installed into <root>/bin.

This is for human/AI inspection, not convention enforcement and not release.

Options:
  --root DIR   Workspace/prefix to use. Defaults to a temporary directory.
  --ref REF    Git ref to checkout for cloned repos. Defaults to main.
  --source-root DIR
               Clone sibling repos from DIR, then overlay dirty/untracked
               worktree files. This keeps local changes and Git metadata while
               still building in an isolated root.
  -h, --help   Show this help.
EOF
}

root=""
ref="main"
source_root=""

while (($# > 0)); do
  case "$1" in
    --root)
      root="${2:-}"
      shift
      ;;
    --ref)
      ref="${2:-}"
      shift
      ;;
    --source-root)
      source_root="${2:-}"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

if [[ -z "$root" ]]; then
  root="$(mktemp -d "${TMPDIR:-/tmp}/jwa-analysis.XXXXXX")"
  echo "temporary root: $root"
fi

mkdir -p "$root/src" "$root/bin"

clone_or_update() {
  local name="$1"
  local dir="$root/src/$name"
  if [[ -n "$source_root" ]]; then
    local src="$source_root/$name"
    if [[ ! -d "$src" ]]; then
      echo "missing local source repo: $src" >&2
      exit 1
    fi
    rm -rf "$dir"
    if git -C "$src" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
      local head
      local origin
      head="$(git -C "$src" rev-parse HEAD)"
      git clone --quiet --no-hardlinks "$src" "$dir"
      git -C "$dir" checkout --quiet --detach "$head"
      if origin="$(git -C "$src" remote get-url origin 2>/dev/null)"; then
        git -C "$dir" remote set-url origin "$origin"
      fi
    else
      mkdir -p "$dir"
    fi
    rsync -a --delete \
      --exclude .git \
      --exclude bin \
      --exclude dist \
      --include .env.template \
      --exclude .env \
      --exclude '.env.*' \
      --exclude .pytest_cache \
      "$src/" "$dir/"
    return
  fi
  if [[ ! -d "$dir/.git" ]]; then
    git clone --filter=blob:none "https://github.com/jwa91/$name.git" "$dir"
  fi
  git -C "$dir" fetch --depth=1 origin "$ref"
  git -C "$dir" checkout --detach FETCH_HEAD
}

build_cli() {
  local name="$1"
  local package="$2"
  (cd "$root/src/$name" && go build -o "$root/bin/$name" "$package")
}

clone_or_update homebrew-tap
clone_or_update jwa-tobrew
clone_or_update jwa-harden
clone_or_update agentskills
clone_or_update prehandover

build_cli jwa-tobrew ./cmd/jwa-tobrew
build_cli jwa-harden ./cmd/jwa-harden
build_cli agentskills ./tools/agentskills
build_cli prehandover ./cmd/prehandover

cat <<EOF

Analysis environment ready.

Root: $root
Source: ${source_root:-github.com/jwa91@$ref}
Binaries:
  $root/bin/jwa-tobrew
  $root/bin/jwa-harden
  $root/bin/agentskills
  $root/bin/prehandover

Use it with:
  export PATH="$root/bin:\$PATH"
  cd "$root/src"

EOF
