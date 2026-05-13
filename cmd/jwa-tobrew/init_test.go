package main

import (
	"os"
	"path/filepath"
	"strings"
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

func TestScaffoldNonGoKinds(t *testing.T) {
	tests := []struct {
		kind            string
		wantRelease     bool
		wantReleaseKind string
	}{
		{kind: "swift-cask", wantRelease: true, wantReleaseKind: "cask"},
		{kind: "cask", wantRelease: true, wantReleaseKind: "cask"},
		{kind: "formula", wantRelease: true, wantReleaseKind: "formula"},
		{kind: "vps"},
	}
	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			dir := t.TempDir()
			bin := t.TempDir()
			writeFakeAgentskills(t, bin)
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

			if err := scaffold(tt.kind, dir, "jwa91", "demo", "demo", "Demo app", false); err != nil {
				t.Fatalf("scaffold: %v", err)
			}

			releasePath := filepath.Join(dir, "scripts", "release.sh")
			if tt.wantRelease {
				body := readFile(t, releasePath)
				if !strings.Contains(body, "--kind "+tt.wantReleaseKind) {
					t.Fatalf("release script missing --kind %s:\n%s", tt.wantReleaseKind, body)
				}
				if info, err := os.Stat(releasePath); err != nil {
					t.Fatalf("release script missing: %v", err)
				} else if info.Mode()&0o111 == 0 {
					t.Fatalf("release script is not executable: %v", info.Mode())
				}
			} else if _, err := os.Stat(releasePath); !os.IsNotExist(err) {
				t.Fatalf("vps scaffold wrote release script, stat err=%v", err)
			}

			agentContract := readFile(t, filepath.Join(dir, "AGENTS.md"))
			if !strings.Contains(agentContract, "jwa-harden run -- <command>") {
				t.Fatalf("AGENTS.md missing jwa-harden guidance:\n%s", agentContract)
			}
			cursorRule := readFile(t, filepath.Join(dir, ".cursor", "rules", "release-contract.mdc"))
			if !strings.Contains(cursorRule, "alwaysApply: true") {
				t.Fatalf("cursor release contract missing alwaysApply frontmatter:\n%s", cursorRule)
			}
			if _, err := os.Stat(filepath.Join(dir, ".agents", "skills", "release", "SKILL.md")); err != nil {
				t.Fatalf("release skill not bootstrapped: %v", err)
			}
			if _, err := os.Stat(filepath.Join(dir, ".agents", "skills", "jwa-harden", "SKILL.md")); err != nil {
				t.Fatalf("jwa-harden skill not bootstrapped: %v", err)
			}
			if _, err := os.Lstat(filepath.Join(dir, ".claude", "skills")); err != nil {
				t.Fatalf("agentskills link did not create harness symlink: %v", err)
			}
		})
	}
}

func TestWriteAgentContractsOverwritePolicy(t *testing.T) {
	dir := t.TempDir()
	agentsPath := filepath.Join(dir, "AGENTS.md")
	rulePath := filepath.Join(dir, ".cursor", "rules", "release-contract.mdc")
	if err := os.MkdirAll(filepath.Dir(rulePath), 0o755); err != nil {
		t.Fatalf("mkdir rule dir: %v", err)
	}
	if err := os.WriteFile(agentsPath, []byte("custom agents\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	if err := os.WriteFile(rulePath, []byte("custom rule\n"), 0o644); err != nil {
		t.Fatalf("write cursor rule: %v", err)
	}

	if err := writeAgentContracts(dir, false); err != nil {
		t.Fatalf("writeAgentContracts false: %v", err)
	}
	if got := readFile(t, agentsPath); got != "custom agents\n" {
		t.Fatalf("AGENTS.md overwritten without force: %q", got)
	}
	if got := readFile(t, rulePath); got != "custom rule\n" {
		t.Fatalf("cursor rule overwritten without force: %q", got)
	}

	if err := writeAgentContracts(dir, true); err != nil {
		t.Fatalf("writeAgentContracts true: %v", err)
	}
	if got := readFile(t, agentsPath); !strings.Contains(got, "jwa-tobrew lint") {
		t.Fatalf("AGENTS.md not overwritten with canonical contract:\n%s", got)
	}
	if got := readFile(t, rulePath); !strings.Contains(got, "jwa-tobrew lint") {
		t.Fatalf("cursor rule not overwritten with canonical contract:\n%s", got)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(body)
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
    mkdir -p "$project/.agents/skills/jwa-harden"
    printf '%s\n' '---' 'name: jwa-harden' '---' > "$project/.agents/skills/jwa-harden/SKILL.md"
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
