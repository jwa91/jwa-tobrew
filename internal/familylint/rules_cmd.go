package familylint

import (
	"fmt"
	"regexp"
	"strings"
)

// versionLineRE matches "<name> <semver|dev> (commit <sha>, built <date>)".
// Local builds report dev; released builds report SemVer.
var versionLineRE = regexp.MustCompile(
	`^\S+\s+(?:dev|\d+\.\d+\.\d+\S*)\s+\(commit\s+\S+,\s+built\s+\S+\)\s*$`,
)

func init() {
	Register(Rule{
		ID:          "F-cmd-001",
		Layer:       LayerCmd,
		Severity:    SeverityFail,
		Description: `"<cli> version" prints "<name> <semver|dev> (commit <sha>, built <date>)"`,
		Check: func(c *Context) Result {
			if c.BinaryPath == "" {
				return Skip("no BinaryPath in Context")
			}
			out := c.RunBinary("version")
			if out.ExecErr != nil {
				return Fail("could not execute binary: "+out.ExecErr.Error(),
					"set Context.BinaryPath to a built binary (try `make build && ./bin/"+c.RepoName+"`)")
			}
			if out.Exit != 0 {
				return Fail(fmt.Sprintf("`%s version` exited %d", c.RepoName, out.Exit),
					"the `version` subcommand should succeed with exit 0")
			}
			line := strings.TrimSpace(out.Stdout)
			if line == "" {
				return Fail("`version` produced no stdout",
					"emit one line: <name> X.Y.Z (commit <sha>, built <date>)")
			}
			if !versionLineRE.MatchString(line) {
				return Fail("version output doesn't match contract: "+line,
					`expected: "<name> X.Y.Z (commit <sha>, built <iso-date>)" or dev during local builds`)
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-cmd-002",
		Layer:       LayerCmd,
		Severity:    SeverityFail,
		Description: `"<cli> help" (and --help, -h, no-args) prints usage to stdout with exit 0`,
		Check: func(c *Context) Result {
			if c.BinaryPath == "" {
				return Skip("no BinaryPath in Context")
			}
			for _, args := range [][]string{nil, {"help"}, {"--help"}, {"-h"}} {
				out := c.RunBinary(args...)
				if out.ExecErr != nil {
					return Fail("exec error on `"+strings.Join(args, " ")+"`: "+out.ExecErr.Error(), "")
				}
				if out.Exit != 0 {
					return Fail(fmt.Sprintf("`%s %s` exited %d (expected 0)",
						c.RepoName, strings.Join(args, " "), out.Exit),
						"help must succeed; usage isn't an error")
				}
				if strings.TrimSpace(out.Stdout) == "" {
					return Fail(fmt.Sprintf("`%s %s` wrote nothing to stdout",
						c.RepoName, strings.Join(args, " ")),
						"emit usage to stdout (not stderr)")
				}
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-cmd-003",
		Layer:       LayerCmd,
		Severity:    SeverityFail,
		Description: `"<cli> doctor" subcommand exists`,
		Check: func(c *Context) Result {
			if c.BinaryPath == "" {
				return Skip("no BinaryPath in Context")
			}
			out := c.RunBinary("doctor")
			if out.ExecErr != nil {
				return Fail("exec error: "+out.ExecErr.Error(), "")
			}
			// Unknown subcommand → exit 2 (per F-cmd-005). Anything else =
			// "doctor exists and ran" (it may exit 4 if checks fail, that's
			// the subcommand's prerogative).
			if out.Exit == 2 {
				return Fail("`doctor` subcommand not recognised (exit 2)",
					"add a `doctor` subcommand that reports runtime preconditions")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-cmd-004",
		Layer:       LayerCmd,
		Severity:    SeverityFail,
		Description: `"jwa-tobrew align" owns repo convention alignment`,
		Check: func(c *Context) Result {
			if c.BinaryPath == "" {
				return Skip("no BinaryPath in Context")
			}
			if c.RepoName != "jwa-tobrew" {
				return Skip("align is intentionally scoped to jwa-tobrew")
			}
			out := c.RunBinary("align")
			if out.ExecErr != nil {
				return Fail("exec error: "+out.ExecErr.Error(), "")
			}
			if out.Exit == 2 {
				return Fail("`align` subcommand not recognised (exit 2)",
					"add an `align` subcommand that checks this CLI's repo-side conventions")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-cmd-005",
		Layer:       LayerCmd,
		Severity:    SeverityFail,
		Description: `Unknown subcommand exits 2 with diagnostics on stderr`,
		Check: func(c *Context) Result {
			if c.BinaryPath == "" {
				return Skip("no BinaryPath in Context")
			}
			// A name that's vanishingly unlikely to be a real subcommand.
			out := c.RunBinary("__definitely_not_a_real_subcommand_xyz__")
			if out.ExecErr != nil {
				return Fail("exec error: "+out.ExecErr.Error(), "")
			}
			if out.Exit != 2 {
				return Fail(fmt.Sprintf("unknown subcommand exited %d, want 2", out.Exit),
					"return exit 2 for usage errors (unknown subcommand, bad flag, etc.)")
			}
			if strings.TrimSpace(out.Stderr) == "" {
				return Fail("unknown subcommand wrote nothing to stderr",
					"diagnostics belong on stderr; usage hint also welcome")
			}
			if strings.TrimSpace(out.Stdout) != "" {
				return Warn("unknown subcommand also wrote to stdout — prefer stderr-only diagnostics")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-cmd-006",
		Layer:       LayerCmd,
		Severity:    SeverityFail,
		Description: `Every known subcommand accepts -h/--help with exit 0`,
		Check: func(c *Context) Result {
			if c.BinaryPath == "" {
				return Skip("no BinaryPath in Context")
			}
			subs := discoverSubcommands(c)
			if len(subs) == 0 {
				return Skip("could not discover subcommands from help output")
			}
			for _, sub := range subs {
				out := c.RunBinary(sub, "-h")
				if out.ExecErr != nil {
					return Fail(fmt.Sprintf("exec error on `%s -h`: %s", sub, out.ExecErr.Error()), "")
				}
				if out.Exit != 0 {
					return Fail(fmt.Sprintf("`%s %s -h` exited %d (expected 0)", c.RepoName, sub, out.Exit),
						"every subcommand must handle -h/--help and exit 0")
				}
			}
			return Pass()
		},
	})
}

// discoverSubcommands runs `<cli> --help` and extracts subcommand names from a
// Commands section. It deliberately avoids scraping the whole help output:
// usage lines and examples often start with the binary name, which creates
// false command names and noisy failures.
func discoverSubcommands(c *Context) []string {
	out := c.RunBinary("--help")
	if out.ExecErr != nil || out.Exit != 0 {
		return nil
	}
	body := out.Stdout
	idx := strings.Index(body, "Commands:")
	if idx < 0 {
		idx = strings.Index(body, "Available commands:")
	}
	if idx < 0 {
		return nil
	}

	var commandLines []string
	for _, line := range strings.Split(body[idx:], "\n")[1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if len(commandLines) > 0 {
				break
			}
			continue
		}
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			break
		}
		commandLines = append(commandLines, line)
	}

	subRE := regexp.MustCompile(`^\s+([a-z][a-z0-9_-]+)\b`)
	seen := map[string]struct{}{}
	var out2 []string
	for _, line := range commandLines {
		m := subRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := m[1]
		switch name {
		case "help", "version": // these are special-cased elsewhere
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		out2 = append(out2, name)
	}
	return out2
}
