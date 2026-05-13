package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAuxiliaryUsesAgentsSkillsLayout(t *testing.T) {
	dir := t.TempDir()
	bin := t.TempDir()
	writeFakeAgentskills(t, bin)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := writeAuxiliary(dir, false); err != nil {
		t.Fatalf("writeAuxiliary: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".agents", "skills", "release", "SKILL.md")); err != nil {
		t.Fatalf("release skill not written under .agents/skills: %v", err)
	}

	link := filepath.Join(dir, ".claude", "skills")
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("harness skills symlink missing: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("harness skills link is not a symlink")
	}
	target, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	if want := filepath.Join("..", ".agents", "skills"); target != want {
		t.Fatalf("symlink target = %q, want %q", target, want)
	}
}

func writeFakeAgentskills(t *testing.T, bin string) {
	t.Helper()
	path := filepath.Join(bin, "agentskills")
	body := `#!/bin/sh
set -eu
cmd="${1:?command required}"
shift
project=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --project)
      project="${2:?project required}"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done
case "$cmd" in
  bootstrap)
    mkdir -p "$project/.agents/skills/release"
    printf '%s\n' '---' 'name: release' '---' > "$project/.agents/skills/release/SKILL.md"
    ;;
  link)
    mkdir -p "$project/.claude"
    ln -sfn "../.agents/skills" "$project/.claude/skills"
    ;;
  *)
    echo "unexpected agentskills command: $cmd" >&2
    exit 2
    ;;
esac
`
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake agentskills: %v", err)
	}
}
