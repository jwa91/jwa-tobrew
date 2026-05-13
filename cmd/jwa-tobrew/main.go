package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

type commandSpec struct {
	Name    string
	Args    string
	Summary string
}

var commandSpecs = []commandSpec{
	{Name: "add", Args: "<github-url>", Summary: "Snapshot a published GitHub release into the tap (no token needed)"},
	{Name: "align", Summary: "Report (or apply) repo drift from current jwa-tobrew conventions"},
	{Name: "bump", Args: "<name> [ver]", Summary: "Re-sync an existing Cask or Formula to a published GitHub release"},
	{Name: "completion", Args: "<shell>", Summary: "Generate shell completion for bash, zsh, or fish"},
	{Name: "config", Summary: "Regenerate tap.toml + tap.local.toml from current tap state"},
	{Name: "deps", Summary: "Show dependency overview for every item in the tap"},
	{Name: "doctor", Summary: "Check tools, tap location, SSH origin, and required env"},
	{Name: "init", Summary: "Scaffold release config in the current project (Go binary or macOS cask)"},
	{Name: "lint", Summary: "Run the jwa-* family policy lint rules"},
	{Name: "release", Summary: "Tag, create GitHub release, and update the tap (run inside a project repo)"},
	{Name: "upgrade", Summary: "Re-install jwa-tobrew via brew"},
	{Name: "version", Summary: "Print build info"},
}

func usageText() string {
	var b strings.Builder
	b.WriteString(`jwa-tobrew — release a project to your personal Homebrew tap

Usage:
  jwa-tobrew <command> [flags]

Commands:
`)
	for _, cmd := range commandSpecs {
		left := cmd.Name
		if cmd.Args != "" {
			left += " " + cmd.Args
		}
		fmt.Fprintf(&b, "  %-23s %s\n", left, cmd.Summary)
	}
	b.WriteString(`
Run 'jwa-tobrew <command> -h' for command-specific flags.

Secret handling: jwa-tobrew expects $GITHUB_TOKEN in env when it hits the GitHub
API (release flow). Wrap with: jwa-harden run -- jwa-tobrew <command>
See ~/dotfiles/docs/security-ground-rules.md for the model.
`)
	return b.String()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usageText())
		return
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "add":
		err = runAdd(args)
	case "align":
		err = runAlign(args)
	case "bump":
		err = runBump(args)
	case "completion":
		err = runCompletion(args)
	case "config":
		err = runConfig(args)
	case "deps":
		err = runDeps(args)
	case "doctor":
		err = runDoctor(args)
	case "init":
		err = runInit(args)
	case "lint":
		err = runLint(args)
	case "release":
		err = runRelease(args)
	case "upgrade":
		err = runUpgrade(args)
	case "version", "-v", "--version":
		fmt.Printf("jwa-tobrew %s (commit %s, built %s)\n", version, commit, date)
		return
	case "-h", "--help", "help":
		fmt.Print(usageText())
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %q\n\n%s", cmd, usageText())
		os.Exit(2)
	}
	if err != nil {
		var exit exitCodeError
		if errors.As(err, &exit) {
			os.Exit(exit.code)
		}
		fmt.Fprintf(os.Stderr, "%s error: %v\n", cmd, err)
		os.Exit(1)
	}
}

type exitCodeError struct {
	code int
}

func (e exitCodeError) Error() string {
	return fmt.Sprintf("exit %d", e.code)
}

// Set by GoReleaser via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// subFlagSet returns a flag set that prints command-specific help.
func subFlagSet(name, summary string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "jwa-tobrew %s — %s\n\nFlags:\n", name, summary)
		fs.PrintDefaults()
	}
	return fs
}

// parseFlags reorders args so flags can appear after positionals, then parses.
// Go's stdlib flag package stops at the first non-flag token, which would
// reject e.g. `jwa-tobrew add github.com/x/y --kind=cask`; this shim makes
// such interleaved usage work the way most CLIs do.
//
// For each "-foo" token, it consults the FlagSet to learn whether the flag
// takes a value; if so, the next token is treated as that value (unless the
// flag was passed as "--foo=val"). Anything after "--" is a literal positional.
func parseFlags(fs *flag.FlagSet, args []string) error {
	isBool := func(name string) bool {
		f := fs.Lookup(name)
		if f == nil {
			return false
		}
		if bf, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && bf.IsBoolFlag() {
			return true
		}
		return false
	}
	var flags, pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			pos = append(pos, args[i+1:]...)
			break
		}
		if strings.HasPrefix(a, "-") && a != "-" {
			flags = append(flags, a)
			name := strings.TrimLeft(a, "-")
			if eq := strings.IndexByte(name, '='); eq >= 0 {
				continue
			}
			if !isBool(name) && i+1 < len(args) {
				flags = append(flags, args[i+1])
				i++
			}
		} else {
			pos = append(pos, a)
		}
	}
	return fs.Parse(append(flags, pos...))
}
