package familylint_test

import (
	"testing"

	"github.com/jwa91/jwa-tobrew/internal/familylint"
)

const noCommandTableCLI = `#!/bin/sh
case "$1" in
  ""|help|--help|-h)
    printf 'Usage:\n  demo <command>\n\nRun demo help <command> for details.\n'
    exit 0
    ;;
  version)
    printf 'demo dev (commit none, built unknown)\n'
    exit 0
    ;;
  doctor)
    printf '! demo doctor\n'
    exit 0
    ;;
  *)
    printf 'unknown command: %s\n' "$1" >&2
    exit 2
    ;;
esac
`

func TestCmdRule004AlignScopedToJwaTobrew(t *testing.T) {
	t.Parallel()

	f := newRepoFixture(t, "demo")
	bin := f.writeExecutable("bin/demo", noCommandTableCLI)

	got := f.runRule("F-cmd-004", familylint.WithBinary(bin)).Status
	if got != familylint.StatusSkip {
		t.Errorf("status = %v, want %v", got, familylint.StatusSkip)
	}
}

func TestCmdRule006DoesNotScrapeUsageAsCommandTable(t *testing.T) {
	t.Parallel()

	f := newRepoFixture(t, "demo")
	bin := f.writeExecutable("bin/demo", noCommandTableCLI)

	got := f.runRule("F-cmd-006", familylint.WithBinary(bin)).Status
	if got != familylint.StatusSkip {
		t.Errorf("status = %v, want %v", got, familylint.StatusSkip)
	}
}

func TestIORule005SkipsWhenLintIsNotAdvertised(t *testing.T) {
	t.Parallel()

	f := newRepoFixture(t, "demo")
	bin := f.writeExecutable("bin/demo", noCommandTableCLI)

	got := f.runRule("F-io-005", familylint.WithBinary(bin)).Status
	if got != familylint.StatusSkip {
		t.Errorf("status = %v, want %v", got, familylint.StatusSkip)
	}
}
