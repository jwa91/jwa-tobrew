package familylint

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func init() {
	Register(Rule{
		ID:          "F-repo-001",
		Layer:       LayerRepo,
		Severity:    SeverityFail,
		Description: `cmd/<repo-name>/main.go exists`,
		Check: func(c *Context) Result {
			rel := filepath.Join("cmd", c.RepoName, "main.go")
			if !c.FileExists(rel) {
				return Fail("missing "+rel,
					"create the canonical Go-CLI layout: cmd/"+c.RepoName+"/main.go with package main")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-repo-002",
		Layer:       LayerRepo,
		Severity:    SeverityFail,
		Description: `Version vars exposed either via internal/version package or package main vars (consistent within a repo)`,
		Check: func(c *Context) Result {
			hasInternal := c.FileExists("internal/version/version.go")
			mainPath := filepath.Join("cmd", c.RepoName, "main.go")
			mainBody, err := c.ReadFile(mainPath)
			hasMainVars := false
			if err == nil {
				body := string(mainBody)
				// Crude but adequate: look for the three var names co-located
				// with `= "dev"`, `= "none"`, `= "unknown"`.
				hasMainVars = strings.Contains(body, `version`) &&
					strings.Contains(body, `commit`) &&
					strings.Contains(body, `date`) &&
					strings.Contains(body, `"dev"`)
			}
			switch {
			case hasInternal && hasMainVars:
				return Warn("both internal/version and package-main vars present; pick one and remove the other")
			case !hasInternal && !hasMainVars:
				return Fail("no version vars found",
					"either internal/version/version.go OR `var version,commit,date` in package main; goreleaser ldflags must target the same path")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-repo-003",
		Layer:       LayerRepo,
		Severity:    SeverityFail,
		Description: `go.mod at root, module path github.com/jwa91/<repo>`,
		Check: func(c *Context) Result {
			body, err := c.ReadFile("go.mod")
			if err != nil {
				return Fail("go.mod not readable: "+err.Error(),
					"run `go mod init github.com/jwa91/"+c.RepoName+"`")
			}
			want := "module github.com/jwa91/" + c.RepoName
			if !strings.Contains(string(body), want) {
				return Fail("go.mod module path doesn't match "+want,
					"adjust module line to "+want+" or rename the repo")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-repo-004",
		Layer:       LayerRepo,
		Severity:    SeverityFail,
		Description: `Makefile present with build/install/check/release targets`,
		Check: func(c *Context) Result {
			if _, err := c.Makefile(); err != nil {
				return Fail("Makefile missing or unreadable: "+err.Error(),
					"add a Makefile with build, install, check, release targets")
			}
			targets, err := c.MakefileTargets()
			if err != nil {
				return Fail("could not parse Makefile: "+err.Error(), "")
			}
			required := []string{"build", "install", "check", "release"}
			var missing []string
			for _, t := range required {
				if _, ok := targets[t]; !ok {
					missing = append(missing, t)
				}
			}
			if len(missing) > 0 {
				return Fail("Makefile missing targets: "+strings.Join(missing, ", "),
					"add the missing targets; see jwa-tobrew/Makefile for the canonical shape")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-repo-005",
		Layer:       LayerRepo,
		Severity:    SeverityFail,
		Description: `README.md present with "brew install jwa91/tap/<name>" line`,
		Check: func(c *Context) Result {
			body, err := c.README()
			if err != nil {
				return Fail("README.md missing: "+err.Error(),
					"add README.md including a `brew install jwa91/tap/"+c.RepoName+"` example")
			}
			install := "brew install jwa91/tap/" + c.RepoName
			if !strings.Contains(string(body), install) {
				return Fail("README has no "+install+" line",
					"add the canonical install line so the README documents the brew path")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-repo-006",
		Layer:       LayerRepo,
		Severity:    SeverityFail,
		Description: `CHANGELOG.md follows Keep-a-Changelog (header + [Unreleased] section)`,
		Check: func(c *Context) Result {
			cl, err := c.Changelog()
			if err != nil {
				return Fail("CHANGELOG.md missing or unreadable: "+err.Error(),
					"start with `# Changelog` and a `## [Unreleased]` section")
			}
			if !cl.HasHeader {
				return Fail("CHANGELOG.md doesn't start with `# Changelog`",
					"first non-blank line must be `# Changelog`")
			}
			hasUnreleased := false
			for _, s := range cl.Sections {
				if strings.EqualFold(s.Version, "Unreleased") {
					hasUnreleased = true
					break
				}
			}
			if !hasUnreleased {
				return Fail("no `## [Unreleased]` section",
					"keep an empty `## [Unreleased]` above the latest version so new entries have a home")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-repo-007",
		Layer:       LayerRepo,
		Severity:    SeverityFail,
		Description: `.goreleaser.yaml present (deep checks in F-cfg-*)`,
		Check: func(c *Context) Result {
			if _, err := c.Goreleaser(); err != nil {
				if os.IsNotExist(err) {
					return Fail(".goreleaser.yaml missing",
						"add the goreleaser config; see jwa-tobrew/.goreleaser.yaml for the canonical shape")
				}
				return Fail(".goreleaser.yaml unreadable: "+err.Error(), "")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-repo-008",
		Layer:       LayerRepo,
		Severity:    SeverityFail,
		Description: `.github/workflows/release.yml present`,
		Check: func(c *Context) Result {
			if _, err := c.ReleaseWorkflow(); err != nil {
				if os.IsNotExist(err) {
					return Fail(".github/workflows/release.yml missing",
						"add a release workflow (manual or tag-triggered)")
				}
				return Fail("release.yml unreadable: "+err.Error(), "")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-repo-009",
		Layer:       LayerRepo,
		Severity:    SeverityFail,
		Description: `.env.template present iff .goreleaser.yaml references op:// URIs`,
		Check: func(c *Context) Result {
			grBody, err := c.ReadFile(".goreleaser.yaml")
			if err != nil {
				grBody, err = c.ReadFile(".goreleaser.yml")
				if err != nil {
					return Skip("no .goreleaser.yaml")
				}
			}
			needsTemplate := strings.Contains(string(grBody), "op://")
			hasTemplate := c.FileExists(".env.template")
			switch {
			case needsTemplate && !hasTemplate:
				return Fail(".goreleaser.yaml references op:// but .env.template is missing",
					"add a .env.template enumerating each op:// reference")
			case !needsTemplate && hasTemplate:
				return Warn(".env.template present but .goreleaser.yaml doesn't reference op:// — is the template still needed?")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-repo-010",
		Layer:       LayerRepo,
		Severity:    SeverityFail,
		Description: `.gitignore blocks .env, .env.local, .env.*.local`,
		Check: func(c *Context) Result {
			body, err := c.Gitignore()
			if err != nil {
				return Fail(".gitignore missing or unreadable",
					"add .gitignore with .env, .env.local, .env.*.local entries")
			}
			required := []string{".env", ".env.local", ".env.*.local"}
			lines := strings.Split(string(body), "\n")
			has := func(needle string) bool {
				for _, line := range lines {
					if strings.TrimSpace(line) == needle {
						return true
					}
				}
				return false
			}
			var missing []string
			for _, r := range required {
				if !has(r) {
					missing = append(missing, r)
				}
			}
			if len(missing) > 0 {
				return Fail(".gitignore missing entries: "+strings.Join(missing, ", "),
					"add the standard .env block")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-repo-011",
		Layer:       LayerRepo,
		Severity:    SeverityFail,
		Description: `scripts/{codesign.sh,notarize-darwin.sh} present + executable iff .goreleaser.yaml targets darwin`,
		Check: func(c *Context) Result {
			g, err := c.Goreleaser()
			if err != nil {
				return Skip("no .goreleaser.yaml (F-repo-007 reports this)")
			}
			darwinTarget := false
			for _, b := range g.Builds {
				for _, os := range b.Goos {
					if os == "darwin" {
						darwinTarget = true
					}
				}
			}
			if !darwinTarget {
				return Skip("no darwin build target")
			}
			scripts := []string{"scripts/codesign.sh", "scripts/notarize-darwin.sh"}
			for _, s := range scripts {
				if !c.FileExists(s) {
					return Fail(s+" missing",
						"copy from jwa-tobrew/scripts/ and adapt the project name")
				}
				if !c.FileExecutable(s) {
					return Fail(s+" not executable",
						"chmod +x "+s)
				}
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-repo-012",
		Layer:       LayerRepo,
		Severity:    SeverityFail,
		Description: `LICENSE present and matches the goreleaser homebrew_casks license field`,
		Check: func(c *Context) Result {
			body, err := c.ReadFile("LICENSE")
			if err != nil {
				return Fail("LICENSE missing",
					"add a LICENSE file (MIT is the family default)")
			}
			g, gerr := c.Goreleaser()
			if gerr != nil || len(g.HomebrewCasks) == 0 {
				return Skip("no homebrew_casks block to cross-check against")
			}
			want := g.HomebrewCasks[0].License
			if want == "" {
				return Warn("homebrew_casks[0].license is empty")
			}
			head := string(body)
			if len(head) > 200 {
				head = head[:200]
			}
			if !strings.Contains(head, want) && !strings.Contains(strings.ToUpper(head), strings.ToUpper(want)) {
				return Fail(fmt.Sprintf("LICENSE doesn't appear to contain %q in its head", want),
					"either correct the LICENSE file or the homebrew_casks.license field")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-repo-013",
		Layer:       LayerRepo,
		Severity:    SeverityFail,
		Description: `git origin remote uses SSH (no HTTPS-with-token)`,
		Check: func(c *Context) Result {
			origin, err := c.GitOrigin()
			if err != nil {
				return Fail("git remote origin not configured: "+err.Error(),
					"set origin: git remote add origin git@github.com:jwa91/"+c.RepoName+".git")
			}
			if strings.HasPrefix(origin, "git@") || strings.HasPrefix(origin, "ssh://") {
				return Pass()
			}
			return Fail("origin is "+origin+" (not SSH)",
				"git remote set-url origin git@github.com:jwa91/"+c.RepoName+".git")
		},
	})

	var brewfileRE = regexp.MustCompile(`(?i)(^|/)Brewfile($|\.[A-Za-z0-9_-]+$)`)

	Register(Rule{
		ID:          "F-repo-014",
		Layer:       LayerRepo,
		Severity:    SeverityFail,
		Description: `No Brewfile anywhere in the repo (Brewfiles live only in ~/dotfiles)`,
		Check: func(c *Context) Result {
			var found []string
			walkErr := c.WalkRepo(func(rel string, _ os.FileInfo) error {
				if brewfileRE.MatchString(rel) {
					found = append(found, rel)
				}
				return nil
			})
			if walkErr != nil {
				return Fail("walk error: "+walkErr.Error(), "")
			}
			if len(found) > 0 {
				return Fail("Brewfile(s) found: "+strings.Join(found, ", "),
					"move all Brewfile entries into ~/dotfiles/Brewfile; per-repo Brewfiles are forbidden")
			}
			return Pass()
		},
	})
}
