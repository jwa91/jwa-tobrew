package familylint

import (
	"fmt"
	"regexp"
	"strings"
)

func init() {
	// --- F-cfg-0xx — .goreleaser.yaml ------------------------------------

	Register(Rule{
		ID:          "F-cfg-001",
		Layer:       LayerCfg,
		Severity:    SeverityFail,
		Description: `.goreleaser.yaml has version: 2`,
		Check: func(c *Context) Result {
			g, err := c.Goreleaser()
			if err != nil {
				return Skip("no .goreleaser.yaml")
			}
			if g.Version != 2 {
				return Fail(fmt.Sprintf("version: %d (want 2)", g.Version),
					"set `version: 2` at the top of .goreleaser.yaml")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-cfg-002",
		Layer:       LayerCfg,
		Severity:    SeverityFail,
		Description: `.goreleaser.yaml project_name equals the repo name`,
		Check: func(c *Context) Result {
			g, err := c.Goreleaser()
			if err != nil {
				return Skip("no .goreleaser.yaml")
			}
			if g.ProjectName == "" {
				return Fail("project_name missing",
					"set project_name: "+c.RepoName)
			}
			if g.ProjectName != c.RepoName {
				return Fail(fmt.Sprintf("project_name=%q (want %q)", g.ProjectName, c.RepoName),
					"align project_name with the repo's directory name")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-cfg-003",
		Layer:       LayerCfg,
		Severity:    SeverityFail,
		Description: `builds[0] targets darwin+linux × amd64+arm64`,
		Check: func(c *Context) Result {
			g, err := c.Goreleaser()
			if err != nil {
				return Skip("no .goreleaser.yaml")
			}
			if len(g.Builds) == 0 {
				return Fail("no builds: block",
					"add a builds: entry compiling cmd/<name> for darwin+linux × amd64+arm64")
			}
			b := g.Builds[0]
			needOS := map[string]bool{"darwin": false, "linux": false}
			for _, o := range b.Goos {
				needOS[o] = true
			}
			needArch := map[string]bool{"amd64": false, "arm64": false}
			for _, a := range b.Goarch {
				needArch[a] = true
			}
			var missing []string
			for o, ok := range needOS {
				if !ok {
					missing = append(missing, "goos="+o)
				}
			}
			for a, ok := range needArch {
				if !ok {
					missing = append(missing, "goarch="+a)
				}
			}
			if len(missing) > 0 {
				return Fail("build matrix missing: "+strings.Join(missing, ", "),
					"goos: [darwin, linux], goarch: [amd64, arm64]")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-cfg-004",
		Layer:       LayerCfg,
		Severity:    SeverityFail,
		Description: `builds[0].env includes CGO_ENABLED=0`,
		Check: func(c *Context) Result {
			g, err := c.Goreleaser()
			if err != nil || len(g.Builds) == 0 {
				return Skip("no builds: block")
			}
			for _, env := range g.Builds[0].Env {
				if strings.TrimSpace(env) == "CGO_ENABLED=0" {
					return Pass()
				}
			}
			return Fail("CGO_ENABLED=0 not in builds[0].env",
				"add CGO_ENABLED=0 to env so binaries are statically linked")
		},
	})

	Register(Rule{
		ID:          "F-cfg-005",
		Layer:       LayerCfg,
		Severity:    SeverityFail,
		Description: `builds[0].ldflags injects version + commit + date via -X`,
		Check: func(c *Context) Result {
			g, err := c.Goreleaser()
			if err != nil || len(g.Builds) == 0 {
				return Skip("no builds: block")
			}
			joined := strings.Join(g.Builds[0].Ldflags, " ")
			needs := []string{"version=", "commit=", "date="}
			var missing []string
			for _, n := range needs {
				if !strings.Contains(joined, n) {
					missing = append(missing, n)
				}
			}
			if len(missing) > 0 {
				return Fail("ldflags missing -X *"+strings.Join(missing, "/* / *")+"*",
					`include -X "<pkg>.version={{ .Version }}" + commit + date`)
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-cfg-006",
		Layer:       LayerCfg,
		Severity:    SeverityFail,
		Description: `builds[0].hooks.post invokes scripts/codesign.sh`,
		Check: func(c *Context) Result {
			g, err := c.Goreleaser()
			if err != nil || len(g.Builds) == 0 {
				return Skip("no builds: block")
			}
			// Skip if no darwin target — codesign hook isn't needed.
			darwin := false
			for _, o := range g.Builds[0].Goos {
				if o == "darwin" {
					darwin = true
				}
			}
			if !darwin {
				return Skip("no darwin target; codesign hook not required")
			}
			for _, h := range g.Builds[0].Hooks.Post {
				if strings.Contains(h.Cmd, "scripts/codesign.sh") {
					return Pass()
				}
			}
			return Fail("scripts/codesign.sh not invoked in builds.hooks.post",
				`add a post-build hook: cmd: 'scripts/codesign.sh "{{ .Target }}" "{{ .Path }}"'`)
		},
	})

	Register(Rule{
		ID:          "F-cfg-007",
		Layer:       LayerCfg,
		Severity:    SeverityFail,
		Description: `No deprecated brews: block (ADR 0008 — use homebrew_casks:)`,
		Check: func(c *Context) Result {
			g, err := c.Goreleaser()
			if err != nil {
				return Skip("no .goreleaser.yaml")
			}
			if len(g.Brews) > 0 {
				return Fail("brews: block present",
					"migrate to homebrew_casks: per ADR 0008 in jwa91/homebrew-tap")
			}
			if _, ok := g.Raw["brews"]; ok {
				return Fail("brews: key present (even if list is empty)",
					"remove the brews: key entirely")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-cfg-008",
		Layer:       LayerCfg,
		Severity:    SeverityFail,
		Description: `homebrew_casks[0].repository = {owner: jwa91, name: homebrew-tap, branch: main}`,
		Check: func(c *Context) Result {
			g, err := c.Goreleaser()
			if err != nil || len(g.HomebrewCasks) == 0 {
				return Skip("no homebrew_casks: block")
			}
			r := g.HomebrewCasks[0].Repository
			if r.Owner != "jwa91" || r.Name != "homebrew-tap" || r.Branch != "main" {
				return Fail(fmt.Sprintf("repository=%+v", r),
					"set repository: {owner: jwa91, name: homebrew-tap, branch: main}")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-cfg-009",
		Layer:       LayerCfg,
		Severity:    SeverityFail,
		Description: `homebrew_casks[0].directory = "Casks"`,
		Check: func(c *Context) Result {
			g, err := c.Goreleaser()
			if err != nil || len(g.HomebrewCasks) == 0 {
				return Skip("no homebrew_casks: block")
			}
			if g.HomebrewCasks[0].Directory != "Casks" {
				return Fail(fmt.Sprintf("directory=%q (want \"Casks\")", g.HomebrewCasks[0].Directory),
					"set directory: Casks (Formula/ is empty by design as of tap v0.4.1)")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-cfg-010",
		Layer:       LayerCfg,
		Severity:    SeverityFail,
		Description: `homebrew_casks[0] uses plural "binaries:" not deprecated singular "binary:"`,
		Check: func(c *Context) Result {
			g, err := c.Goreleaser()
			if err != nil || len(g.HomebrewCasks) == 0 {
				return Skip("no homebrew_casks: block")
			}
			ck := g.HomebrewCasks[0]
			if ck.Binary != "" {
				return Fail(`uses singular "binary:" (deprecated in goreleaser v2.12.6)`,
					`replace with binaries: ["`+ck.Binary+`"]`)
			}
			if len(ck.Binaries) == 0 {
				return Fail("neither binary: nor binaries: declared",
					`add binaries: ["`+c.RepoName+`"]`)
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-cfg-011",
		Layer:       LayerCfg,
		Severity:    SeverityFail,
		Description: `homebrew_casks[0].repository.token templated as $HOMEBREW_TAP_GITHUB_TOKEN`,
		Check: func(c *Context) Result {
			g, err := c.Goreleaser()
			if err != nil || len(g.HomebrewCasks) == 0 {
				return Skip("no homebrew_casks: block")
			}
			tok := g.HomebrewCasks[0].Repository.Token
			want := "HOMEBREW_TAP_GITHUB_TOKEN"
			if !strings.Contains(tok, want) {
				return Fail(`token=`+tok,
					`set token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"`)
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-cfg-012",
		Layer:       LayerCfg,
		Severity:    SeverityWarn,
		Description: `homebrew_casks[0].license matches LICENSE file head`,
		Check: func(c *Context) Result {
			g, err := c.Goreleaser()
			if err != nil || len(g.HomebrewCasks) == 0 {
				return Skip("no homebrew_casks: block")
			}
			lic := g.HomebrewCasks[0].License
			if lic == "" {
				return Warn("homebrew_casks[0].license is empty")
			}
			body, lerr := c.ReadFile("LICENSE")
			if lerr != nil {
				return Skip("no LICENSE file (F-repo-012 catches this)")
			}
			head := string(body)
			if len(head) > 300 {
				head = head[:300]
			}
			if !strings.Contains(strings.ToUpper(head), strings.ToUpper(lic)) {
				return Warn(fmt.Sprintf("LICENSE head doesn't mention %q — manual check advised", lic))
			}
			return Pass()
		},
	})

	// --- F-cfg-02x — .github/workflows/release.yml -----------------------

	Register(Rule{
		ID:          "F-cfg-020",
		Layer:       LayerCfg,
		Severity:    SeverityFail,
		Description: `release workflow has on.workflow_dispatch`,
		Check: func(c *Context) Result {
			body, err := c.ReleaseWorkflow()
			if err != nil {
				return Skip("no release.yml (F-repo-008 catches this)")
			}
			s := string(body)
			if !strings.Contains(s, "workflow_dispatch:") && !strings.Contains(s, "workflow_dispatch :") {
				return Fail("workflow_dispatch trigger missing",
					"add `workflow_dispatch:` under on: so the workflow can be manually fired")
			}
			return Pass()
		},
	})

	// Pinned-by-SHA pattern: `uses: <owner>/<repo>@<40-hex-chars>`.
	// We flag any `uses:` line whose ref looks like a non-SHA (tag or branch).
	var usesShaRE = regexp.MustCompile(`uses:\s*([^@\s]+)@([^\s#]+)`)

	Register(Rule{
		ID:          "F-cfg-021",
		Layer:       LayerCfg,
		Severity:    SeverityWarn,
		Description: `Third-party actions pinned by 40-char SHA (not by tag)`,
		Check: func(c *Context) Result {
			body, err := c.ReleaseWorkflow()
			if err != nil {
				return Skip("no release.yml")
			}
			var unpinned []string
			for _, m := range usesShaRE.FindAllStringSubmatch(string(body), -1) {
				action, ref := m[1], m[2]
				if strings.HasPrefix(action, "./") || strings.HasPrefix(action, "actions/") &&
					strings.Count(action, "/") == 1 {
					// First-party Actions org and local actions — still need pinning,
					// don't special-case.
				}
				if len(ref) != 40 || !isHex(ref) {
					unpinned = append(unpinned, action+"@"+ref)
				}
			}
			if len(unpinned) > 0 {
				return Warn("unpinned `uses:` refs: " + strings.Join(unpinned, ", "))
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-cfg-022",
		Layer:       LayerCfg,
		Severity:    SeverityFail,
		Description: `release workflow declares permissions explicitly (no implicit default)`,
		Check: func(c *Context) Result {
			body, err := c.ReleaseWorkflow()
			if err != nil {
				return Skip("no release.yml")
			}
			if !strings.Contains(string(body), "permissions:") {
				return Fail("no `permissions:` key in release workflow",
					"add `permissions: contents: read` at workflow level + elevate per-job as needed")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-cfg-023",
		Layer:       LayerCfg,
		Severity:    SeverityFail,
		Description: `Release workflow does not reference long-lived signing secrets in env (safe-CI design)`,
		Check: func(c *Context) Result {
			body, err := c.ReleaseWorkflow()
			if err != nil {
				return Skip("no release.yml")
			}
			s := string(body)
			banned := []string{
				"MACOS_SIGN_P12",
				"MACOS_SIGN_PASSWORD",
				"MACOS_NOTARY_KEY",
				"MACOS_SIGN_IDENTITY",
			}
			var seen []string
			for _, b := range banned {
				if strings.Contains(s, b) {
					seen = append(seen, b)
				}
			}
			if len(seen) > 0 {
				return Fail("signing secrets referenced in workflow: "+strings.Join(seen, ", "),
					"move signing to local `make release`; CI does build-only per ADR-tbd safe-CI design")
			}
			return Pass()
		},
	})

	// --- F-cfg-03x — Makefile --------------------------------------------

	Register(Rule{
		ID:          "F-cfg-030",
		Layer:       LayerCfg,
		Severity:    SeverityFail,
		Description: `Makefile release preflight checks op signin + gh auth + notarytool profile`,
		Check: func(c *Context) Result {
			body, err := c.Makefile()
			if err != nil {
				return Skip("no Makefile")
			}
			s := string(body)
			needs := map[string]string{
				"op whoami":            "op whoami",
				"gh auth":              "gh auth status",
				"notarytool":           "xcrun notarytool history --keychain-profile notarytool",
			}
			var missing []string
			for label, marker := range needs {
				if !strings.Contains(s, marker) {
					missing = append(missing, label)
				}
			}
			if len(missing) > 0 {
				return Fail("release target missing preflight checks: "+strings.Join(missing, ", "),
					"add the preflight commands as @-prefixed lines that exit non-zero on failure")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-cfg-031",
		Layer:       LayerCfg,
		Severity:    SeverityFail,
		Description: `Makefile release accepts v$(VERSION) anywhere in HEAD's ancestry (not strictly HEAD)`,
		Check: func(c *Context) Result {
			body, err := c.Makefile()
			if err != nil {
				return Skip("no Makefile")
			}
			s := string(body)
			// The strict guard pattern `test "$existing" = "$head"` rejects
			// merge-commit flows. The relaxed form uses git merge-base / --is-ancestor.
			if strings.Contains(s, `"$$existing" = "$$head"`) || strings.Contains(s, `"$existing" = "$head"`) {
				return Fail("release target uses strict tag==HEAD guard",
					"relax to: `git merge-base --is-ancestor v$(VERSION) HEAD` so PR-merged tags pass")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-cfg-032",
		Layer:       LayerCfg,
		Severity:    SeverityFail,
		Description: `Makefile release invokes goreleaser via jwa-harden run OR op run`,
		Check: func(c *Context) Result {
			body, err := c.Makefile()
			if err != nil {
				return Skip("no Makefile")
			}
			s := string(body)
			if strings.Contains(s, "jwa-harden run") || strings.Contains(s, "op run") {
				return Pass()
			}
			return Fail("release target invokes goreleaser without a secret wrapper",
				"wrap with `jwa-harden run -- goreleaser …` (general case) or `op run --env-file=.env.template -- goreleaser …` (bootstrap exception)")
		},
	})

	Register(Rule{
		ID:          "F-cfg-033",
		Layer:       LayerCfg,
		Severity:    SeverityFail,
		Description: `Makefile check target runs at minimum go vet + go test`,
		Check: func(c *Context) Result {
			body, err := c.Makefile()
			if err != nil {
				return Skip("no Makefile")
			}
			s := string(body)
			if !strings.Contains(s, "go vet") || !strings.Contains(s, "go test") {
				return Fail("`check` target doesn't run both `go vet` and `go test`",
					"include both in the recipe (or in dependencies of `check`)")
			}
			return Pass()
		},
	})

	// --- F-cfg-04x — CHANGELOG.md ----------------------------------------

	Register(Rule{
		ID:          "F-cfg-040",
		Layer:       LayerCfg,
		Severity:    SeverityFail,
		Description: `CHANGELOG.md starts with "# Changelog"`,
		Check: func(c *Context) Result {
			cl, err := c.Changelog()
			if err != nil {
				return Skip("no CHANGELOG.md (F-repo-006 catches this)")
			}
			if !cl.HasHeader {
				return Fail("CHANGELOG.md doesn't start with `# Changelog`",
					"first non-blank line must be `# Changelog`")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-cfg-041",
		Layer:       LayerCfg,
		Severity:    SeverityFail,
		Description: `CHANGELOG.md has a [Unreleased] section`,
		Check: func(c *Context) Result {
			cl, err := c.Changelog()
			if err != nil {
				return Skip("no CHANGELOG.md")
			}
			for _, s := range cl.Sections {
				if strings.EqualFold(s.Version, "Unreleased") {
					return Pass()
				}
			}
			return Fail("no `## [Unreleased]` section",
				"add `## [Unreleased]` above the latest version so new entries have a home")
		},
	})

	Register(Rule{
		ID:          "F-cfg-042",
		Layer:       LayerCfg,
		Severity:    SeverityFail,
		Description: `Most recent versioned CHANGELOG section matches the latest v* git tag`,
		Check: func(c *Context) Result {
			cl, err := c.Changelog()
			if err != nil {
				return Skip("no CHANGELOG.md")
			}
			tag, terr := c.LatestTag()
			if terr != nil {
				return Skip("git tag lookup failed: " + terr.Error())
			}
			if tag == "" {
				return Skip("no v* tags yet")
			}
			tagVer := strings.TrimPrefix(tag, "v")
			// First non-Unreleased section is "latest".
			for _, s := range cl.Sections {
				if strings.EqualFold(s.Version, "Unreleased") {
					continue
				}
				// Match exact or version-with-noted-suffix like "0.1.1 (never released)".
				if s.Version == tagVer || strings.HasPrefix(s.Version, tagVer+" ") {
					return Pass()
				}
				return Fail(fmt.Sprintf("latest tag=%s but latest CHANGELOG section=[%s]", tag, s.Version),
					"add a `## ["+tagVer+"]` section, or move the tag to match the latest entry")
			}
			return Fail("CHANGELOG has no versioned sections, but tag "+tag+" exists",
				"add a `## ["+tagVer+"]` section describing the release")
		},
	})

	var allowedSubsections = map[string]bool{
		"Added":      true,
		"Changed":    true,
		"Removed":    true,
		"Fixed":      true,
		"Deprecated": true,
		"Security":   true,
	}

	Register(Rule{
		ID:          "F-cfg-043",
		Layer:       LayerCfg,
		Severity:    SeverityWarn,
		Description: `CHANGELOG H3 subsections use Keep-a-Changelog vocabulary`,
		Check: func(c *Context) Result {
			cl, err := c.Changelog()
			if err != nil {
				return Skip("no CHANGELOG.md")
			}
			var bad []string
			for _, sec := range cl.Sections {
				for _, h3 := range sec.Subsections {
					if _, ok := allowedSubsections[h3]; !ok {
						// Allow trailing "(notes)" style headers used in jwa-harden history.
						base := strings.SplitN(h3, " ", 2)[0]
						if !allowedSubsections[base] {
							bad = append(bad, sec.Version+":"+h3)
						}
					}
				}
			}
			if len(bad) > 0 {
				return Warn("non-canonical H3 subsections: " + strings.Join(bad, ", "))
			}
			return Pass()
		},
	})
}

// isHex reports whether s consists entirely of hex digits.
func isHex(s string) bool {
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return s != ""
}
