package familylint_test

import (
	"os/exec"
	"testing"

	"github.com/jwa91/jwa-tobrew/internal/familylint"
)

func TestVerRule002AcceptsDirtyBuildMetadata(t *testing.T) {
	t.Parallel()

	f := newRepoFixture(t, "demo")
	f.write("README.md", "demo\n")
	git(t, f.root, "init", "-q")
	git(t, f.root, "config", "user.email", "test@example.com")
	git(t, f.root, "config", "user.name", "Test User")
	git(t, f.root, "add", "README.md")
	git(t, f.root, "commit", "-q", "-m", "init")
	git(t, f.root, "tag", "v1.2.3")

	bin := f.writeExecutable("bin/demo", `#!/bin/sh
if [ "$1" = "version" ]; then
  printf 'demo 1.2.3+dirty (commit abcdef0, built 2026-05-13T00:00:00Z)\n'
  exit 0
fi
exit 2
`)

	got := f.runRule("F-ver-002", familylint.WithBinary(bin)).Status
	if got != familylint.StatusPass {
		t.Errorf("status = %v, want %v", got, familylint.StatusPass)
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
