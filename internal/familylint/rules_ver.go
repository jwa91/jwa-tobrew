package familylint

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	semverTagRE       = regexp.MustCompile(`^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$`)
	subcomponentTagRE = regexp.MustCompile(`^[a-z][a-z0-9_-]*\/v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$`)
)

func init() {
	Register(Rule{
		ID:          "F-ver-001",
		Layer:       LayerVer,
		Severity:    SeverityFail,
		Description: `All tags are vX.Y.Z or <component>/vX.Y.Z`,
		Check: func(c *Context) Result {
			tags, err := c.GitTags()
			if err != nil {
				return Skip("git tag listing failed: " + err.Error())
			}
			if len(tags) == 0 {
				return Skip("no v* tags yet")
			}
			var bad []string
			for _, t := range tags {
				if !semverTagRE.MatchString(t) && !subcomponentTagRE.MatchString(t) {
					bad = append(bad, t)
				}
			}
			if len(bad) > 0 {
				return Fail("non-canonical tags: "+strings.Join(bad, ", "),
					`use vX.Y.Z (binary releases) or <component>/vX.Y.Z (per-component, e.g. per-skill)`)
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-ver-002",
		Layer:       LayerVer,
		Severity:    SeverityFail,
		Description: `<cli> version output's semver matches the latest v* tag`,
		Check: func(c *Context) Result {
			if c.BinaryPath == "" {
				return Skip("no BinaryPath in Context")
			}
			tag, err := c.LatestTag()
			if err != nil || tag == "" {
				return Skip("no v* tag to compare against")
			}
			out := c.RunBinary("version")
			if out.ExecErr != nil || out.Exit != 0 {
				return Skip("version subcommand failed (F-cmd-001 catches the cause)")
			}
			want := strings.TrimPrefix(tag, "v")
			fields := strings.Fields(strings.TrimSpace(out.Stdout))
			if len(fields) < 2 {
				return Skip("version output not in canonical shape (F-cmd-001 catches this)")
			}
			gotVer := fields[1]
			if gotVer == "dev" {
				return Skip("local dev build is not expected to match the latest release tag")
			}
			gotBase, _, _ := strings.Cut(gotVer, "+")
			if gotBase != want {
				return Fail(fmt.Sprintf("binary reports %s, latest tag is %s", gotVer, tag),
					"rebuild + reinstall the binary so ldflags pick up the new tag (`make install` or brew upgrade)")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-ver-003",
		Layer:       LayerVer,
		Severity:    SeverityFail,
		Description: `Every CHANGELOG version has a matching v* tag`,
		Check: func(c *Context) Result {
			cl, err := c.Changelog()
			if err != nil {
				return Skip("no CHANGELOG.md")
			}
			tags, err := c.GitTags()
			if err != nil {
				return Skip("git tag lookup failed: " + err.Error())
			}
			tagSet := map[string]struct{}{}
			for _, t := range tags {
				tagSet[strings.TrimPrefix(t, "v")] = struct{}{}
			}
			var missing []string
			for _, s := range cl.Sections {
				if strings.EqualFold(s.Version, "Unreleased") {
					continue
				}
				if strings.Contains(strings.ToLower(s.Date), "never released") {
					continue
				}
				if strings.Contains(strings.ToLower(s.Date), "historical") {
					continue
				}
				ver := strings.SplitN(s.Version, " ", 2)[0] // strip "(never released)" markers
				ver = strings.TrimSuffix(ver, ")")
				if _, ok := tagSet[ver]; !ok {
					missing = append(missing, ver)
				}
			}
			if len(missing) > 0 {
				return Fail("CHANGELOG versions without matching tags: "+strings.Join(missing, ", "),
					"create the missing v* tags or rewrite the CHANGELOG to remove unreleased version headers")
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-ver-004",
		Layer:       LayerVer,
		Severity:    SeverityFail,
		Description: `goreleaser project_name == repo name == binary name`,
		Check: func(c *Context) Result {
			g, err := c.Goreleaser()
			if err != nil {
				return Skip("no .goreleaser.yaml")
			}
			if g.ProjectName != c.RepoName {
				return Fail(fmt.Sprintf("project_name=%q repo=%q", g.ProjectName, c.RepoName),
					"align project_name with the repo directory name")
			}
			if len(g.Builds) > 0 && g.Builds[0].Binary != "" && g.Builds[0].Binary != c.RepoName {
				return Fail(fmt.Sprintf("builds[0].binary=%q repo=%q", g.Builds[0].Binary, c.RepoName),
					"set builds[0].binary to the repo name")
			}
			if len(g.HomebrewCasks) > 0 {
				ck := g.HomebrewCasks[0]
				if ck.Name != c.RepoName {
					return Fail(fmt.Sprintf("homebrew_casks[0].name=%q repo=%q", ck.Name, c.RepoName),
						"set homebrew_casks.name to the repo name")
				}
				for _, b := range ck.Binaries {
					if b != c.RepoName {
						return Fail(fmt.Sprintf("homebrew_casks[0].binaries includes %q (repo=%q)", b, c.RepoName),
							"cask binaries must match the repo name; multi-binary casks are out of scope today")
					}
				}
			}
			return Pass()
		},
	})

	Register(Rule{
		ID:          "F-ver-005",
		Layer:       LayerVer,
		Severity:    SeverityWarn,
		Description: `Prereleases only when goreleaser release.prerelease == "auto"`,
		Check: func(c *Context) Result {
			g, err := c.Goreleaser()
			if err != nil {
				return Skip("no .goreleaser.yaml")
			}
			tags, err := c.GitTags()
			if err != nil {
				return Skip("git tag lookup failed")
			}
			prereleaseTag := regexp.MustCompile(`-`)
			anyPre := false
			for _, t := range tags {
				if prereleaseTag.MatchString(strings.TrimPrefix(t, "v")) {
					anyPre = true
					break
				}
			}
			if !anyPre {
				return Pass()
			}
			if g.Release.Prerelease == "auto" {
				return Pass()
			}
			return Warn(fmt.Sprintf("prerelease tag(s) present but release.prerelease=%q (want \"auto\")", g.Release.Prerelease))
		},
	})
}
