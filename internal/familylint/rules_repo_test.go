package familylint

import (
	"os"
	"path/filepath"
	"testing"
)

// newTempRepo builds a minimal "looks like a jwa-* CLI repo" tree under
// t.TempDir() and returns a Context rooted there. Caller adds/overwrites
// files via writeFile.
func newTempRepo(t *testing.T, name string) *Context {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	c, err := NewContext(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func writeFile(t *testing.T, c *Context, rel, body string) {
	t.Helper()
	abs := filepath.Join(c.RepoRoot, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRepoRule_001_CmdMainGo(t *testing.T) {
	c := newTempRepo(t, "demo")
	rule := RuleByID("F-repo-001")

	// Missing → fail.
	if got := rule.Check(c).Status; got != StatusFail {
		t.Fatalf("missing main.go: got status %d, want fail", got)
	}

	// Present at canonical path → pass.
	writeFile(t, c, "cmd/demo/main.go", "package main\n")
	if got := rule.Check(c).Status; got != StatusPass {
		t.Fatalf("present main.go: got status %d, want pass", got)
	}
}

func TestRepoRule_003_GoModule(t *testing.T) {
	c := newTempRepo(t, "demo")
	rule := RuleByID("F-repo-003")

	if got := rule.Check(c).Status; got != StatusFail {
		t.Fatal("missing go.mod should fail")
	}

	writeFile(t, c, "go.mod", "module github.com/jwa91/demo\n\ngo 1.23\n")
	if got := rule.Check(c).Status; got != StatusPass {
		t.Fatal("correct module path should pass")
	}

	writeFile(t, c, "go.mod", "module github.com/someone-else/demo\n\ngo 1.23\n")
	if got := rule.Check(c).Status; got != StatusFail {
		t.Fatal("wrong module path should fail")
	}
}

func TestRepoRule_010_GitignoreEnvBlock(t *testing.T) {
	c := newTempRepo(t, "demo")
	rule := RuleByID("F-repo-010")

	writeFile(t, c, ".gitignore", "bin/\n")
	if got := rule.Check(c).Status; got != StatusFail {
		t.Fatal("missing .env block should fail")
	}

	// Invalidate the lazy cache before re-checking the second body.
	c.gitignore, c.gitignoreErr = nil, nil
	writeFile(t, c, ".gitignore", "bin/\n.env\n.env.local\n.env.*.local\n")
	if got := rule.Check(c).Status; got != StatusPass {
		t.Fatal("complete .env block should pass")
	}
}

func TestCfgRule_007_NoBrews(t *testing.T) {
	c := newTempRepo(t, "demo")
	rule := RuleByID("F-cfg-007")

	// No .goreleaser.yaml → skip.
	if got := rule.Check(c).Status; got != StatusSkip {
		t.Fatalf("no goreleaser file: got status %d, want skip", got)
	}

	// brews: present → fail.
	writeFile(t, c, ".goreleaser.yaml", `
version: 2
project_name: demo
brews:
  - name: demo
`)
	// Reset cache for the new file.
	c.goreleaser, c.goreleaserErr = nil, nil
	if got := rule.Check(c); got.Status != StatusFail {
		t.Fatalf("brews present: got status %d, want fail", got.Status)
	}

	// Only homebrew_casks: → pass.
	writeFile(t, c, ".goreleaser.yaml", `
version: 2
project_name: demo
homebrew_casks:
  - name: demo
    directory: Casks
    binaries: [demo]
`)
	c.goreleaser, c.goreleaserErr = nil, nil
	if got := rule.Check(c); got.Status != StatusPass {
		t.Fatalf("only homebrew_casks: got status %d, want pass", got.Status)
	}
}

func TestCfgRule_010_BinariesPlural(t *testing.T) {
	c := newTempRepo(t, "demo")
	rule := RuleByID("F-cfg-010")

	writeFile(t, c, ".goreleaser.yaml", `
version: 2
project_name: demo
homebrew_casks:
  - name: demo
    binary: demo
`)
	c.goreleaser, c.goreleaserErr = nil, nil
	if got := rule.Check(c); got.Status != StatusFail {
		t.Fatalf("singular binary: should fail, got %d", got.Status)
	}

	writeFile(t, c, ".goreleaser.yaml", `
version: 2
project_name: demo
homebrew_casks:
  - name: demo
    binaries: [demo]
`)
	c.goreleaser, c.goreleaserErr = nil, nil
	if got := rule.Check(c); got.Status != StatusPass {
		t.Fatalf("plural binaries: should pass, got %d", got.Status)
	}
}
