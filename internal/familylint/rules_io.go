package familylint

import "strings"

// The IO layer defines exit-code semantics and diagnostic conventions.
// Most rules here are EMERGENT — proven by the cmd-layer rules that
// already exercise the binary in the relevant way. Where the convention
// can only be observed via a specific provocation, the rule provokes
// directly. Where coverage is fully transitive, the rule Skips with a
// pointer to the test that does cover it.

func init() {
	Register(Rule{
		ID:          "F-io-001",
		Layer:       LayerIO,
		Severity:    SeverityFail,
		Description: `Exit 0 means success`,
		Check: func(c *Context) Result {
			if c.BinaryPath == "" {
				return Skip("no BinaryPath in Context")
			}
			out := c.RunBinary("version")
			if out.ExecErr != nil {
				return Fail("could not execute binary: "+out.ExecErr.Error(), "")
			}
			if out.Exit != 0 {
				return Fail("successful version command exited non-zero",
					"successful commands must exit 0")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-io-003",
		Layer:       LayerIO,
		Severity:    SeverityFail,
		Description: `Exit 2 means usage error (unknown subcommand, bad flag)`,
		Check: func(c *Context) Result {
			if c.BinaryPath == "" {
				return Skip("no BinaryPath in Context")
			}
			out := c.RunBinary("__definitely_not_a_real_subcommand_xyz__")
			if out.Exit != 2 {
				return Fail("unknown subcommand did not exit 2",
					"usage errors must exit 2")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-io-005",
		Layer:       LayerIO,
		Severity:    SeverityFail,
		Description: `Exit 4 is reserved for lint/alignment drift when a lint command exists`,
		Check: func(c *Context) Result {
			if c.BinaryPath == "" {
				return Skip("no BinaryPath in Context")
			}
			subs := discoverSubcommands(c)
			hasLint := false
			for _, sub := range subs {
				if sub == "lint" {
					hasLint = true
					break
				}
			}
			if !hasLint {
				return Skip("lint subcommand not advertised by this CLI")
			}
			out := c.RunBinary("lint", "--unknown-flag")
			if out.ExecErr != nil {
				return Skip("lint subcommand not executable: " + out.ExecErr.Error())
			}
			if out.Exit != 2 {
				return Fail("bad lint flag did not exit 2",
					"lint failures reserve exit 4; usage errors still exit 2")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-io-006",
		Layer:       LayerIO,
		Severity:    SeverityFail,
		Description: `Diagnostics go to stderr, requested output to stdout`,
		Check: func(c *Context) Result {
			// Exercised on the unknown-subcommand path: stderr must contain
			// the diagnostic, stdout should be empty.
			if c.BinaryPath == "" {
				return Skip("no BinaryPath in Context")
			}
			out := c.RunBinary("__definitely_not_a_real_subcommand_xyz__")
			if strings.TrimSpace(out.Stdout) != "" && strings.TrimSpace(out.Stderr) == "" {
				return Fail("error went to stdout, not stderr",
					"diagnostics on stderr; stdout reserved for requested output")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-io-007",
		Layer:       LayerIO,
		Severity:    SeverityWarn,
		Description: `Status lines use the ✓ / ! / ✗ marker convention`,
		Check: func(c *Context) Result {
			if c.BinaryPath == "" {
				return Skip("no BinaryPath in Context")
			}
			// `doctor` is the canonical place to see status markers.
			out := c.RunBinary("doctor")
			if out.ExecErr != nil || out.Exit == 2 {
				return Skip("doctor subcommand not present (covered by F-cmd-003)")
			}
			combined := out.Stdout + out.Stderr
			if !strings.ContainsAny(combined, "✓!✗") {
				return Warn("doctor output uses no ✓/!/✗ markers; convention is to prefix each check line")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-io-008",
		Layer:       LayerIO,
		Severity:    SeverityFail,
		Description: `ANSI colour is suppressed when stdout is not a TTY (and when NO_COLOR is set)`,
		Check: func(c *Context) Result {
			if c.BinaryPath == "" {
				return Skip("no BinaryPath in Context")
			}
			// RunBinary captures via pipes (not TTY) AND sets NO_COLOR=1.
			// If we still see ESC bytes, the binary is ignoring both signals.
			out := c.RunBinary("doctor")
			if out.ExecErr != nil || out.Exit == 2 {
				return Skip("doctor subcommand not present")
			}
			if strings.ContainsRune(out.Stdout+out.Stderr, 0x1b) {
				return Fail("output contained ESC byte under NO_COLOR=1 + non-TTY",
					"gate ANSI escapes on isatty(stdout) && os.Getenv(\"NO_COLOR\")==\"\"")
			}
			return Pass()
		},
	})
}
