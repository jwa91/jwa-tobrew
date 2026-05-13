package familylint

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// --- .goreleaser.yaml ------------------------------------------------------

// Goreleaser is the subset of goreleaser config that family rules inspect.
// Fields we deeply assert on are typed; the rest sits in Raw for lookups
// we'd rather not hard-code yet.
type Goreleaser struct {
	Path          string           `yaml:"-"`
	Version       int              `yaml:"version"`
	ProjectName   string           `yaml:"project_name"`
	Builds        []GoreleaserBuild `yaml:"builds"`
	Brews         []map[string]any `yaml:"brews"` // any presence is a fail (deprecated)
	HomebrewCasks []GoreleaserCask `yaml:"homebrew_casks"`
	Release       GoreleaserRelease `yaml:"release"`
	Raw           map[string]any   `yaml:"-"`
}

// GoreleaserBuild is one entry under builds:.
type GoreleaserBuild struct {
	ID      string             `yaml:"id"`
	Main    string             `yaml:"main"`
	Binary  string             `yaml:"binary"`
	Goos    []string           `yaml:"goos"`
	Goarch  []string           `yaml:"goarch"`
	Env     []string           `yaml:"env"`
	Ldflags []string           `yaml:"ldflags"`
	Hooks   GoreleaserHooks    `yaml:"hooks"`
}

// GoreleaserHooks holds build-stage pre/post hooks.
type GoreleaserHooks struct {
	Pre  []GoreleaserHook `yaml:"pre"`
	Post []GoreleaserHook `yaml:"post"`
}

// GoreleaserHook accepts both the scalar form ("script.sh foo") and the
// mapping form ({cmd: "script.sh", dir: ".", ...}).
type GoreleaserHook struct {
	Cmd string `yaml:"cmd"`
}

// UnmarshalYAML supports both scalar and mapping representations.
func (h *GoreleaserHook) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		h.Cmd = node.Value
		return nil
	case yaml.MappingNode:
		type plain GoreleaserHook
		return node.Decode((*plain)(h))
	default:
		return fmt.Errorf("goreleaser hook: unexpected yaml kind %d", node.Kind)
	}
}

// GoreleaserCask is one homebrew_casks entry.
type GoreleaserCask struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	Homepage    string          `yaml:"homepage"`
	License     string          `yaml:"license"`
	Directory   string          `yaml:"directory"`
	Binaries    []string        `yaml:"binaries"` // plural, modern
	Binary      string          `yaml:"binary"`   // singular, deprecated — flag if present
	Repository  GoreleaserRepo  `yaml:"repository"`
	Dependencies []GoreleaserDep `yaml:"dependencies"`
}

// GoreleaserRepo describes the target tap.
type GoreleaserRepo struct {
	Owner  string `yaml:"owner"`
	Name   string `yaml:"name"`
	Branch string `yaml:"branch"`
	Token  string `yaml:"token"`
}

// GoreleaserDep is one item in homebrew_casks[].dependencies.
type GoreleaserDep struct {
	Formula string `yaml:"formula"`
	Cask    string `yaml:"cask"`
}

// GoreleaserRelease holds release-stage settings.
type GoreleaserRelease struct {
	Draft      bool   `yaml:"draft"`
	Prerelease string `yaml:"prerelease"` // "auto" | "true" | "false"
}

// Goreleaser returns the parsed config, cached. Returns the file's read
// error (which may be os.ErrNotExist) so callers can decide whether
// absence is "skip" or "fail".
func (c *Context) Goreleaser() (*Goreleaser, error) {
	if c.goreleaser != nil || c.goreleaserErr != nil {
		return c.goreleaser, c.goreleaserErr
	}
	path := filepath.Join(c.RepoRoot, ".goreleaser.yaml")
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		alt := filepath.Join(c.RepoRoot, ".goreleaser.yml")
		body, err = os.ReadFile(alt)
		if err == nil {
			path = alt
		}
	}
	if err != nil {
		c.goreleaserErr = err
		return nil, err
	}
	g := &Goreleaser{Path: path}
	if err := yaml.Unmarshal(body, g); err != nil {
		c.goreleaserErr = fmt.Errorf("parse %s: %w", path, err)
		return nil, c.goreleaserErr
	}
	if err := yaml.Unmarshal(body, &g.Raw); err != nil {
		c.goreleaserErr = fmt.Errorf("parse %s raw: %w", path, err)
		return nil, c.goreleaserErr
	}
	c.goreleaser = g
	return g, nil
}

// --- CHANGELOG.md ----------------------------------------------------------

// Changelog is a light parse of Keep-a-Changelog files. Sections is in
// document order: the [Unreleased] entry (if present) is at index 0.
type Changelog struct {
	Path        string
	HasHeader   bool             // line 1 (skipping blanks) is "# Changelog"
	Sections    []ChangelogSection
}

// ChangelogSection is one ## [version] entry.
type ChangelogSection struct {
	Version    string   // e.g. "Unreleased", "0.4.2"
	Date       string   // raw date string (or "")
	Subsections []string // ordered list of H3 names ("Added", "Changed", ...)
}

var changelogVersionLine = regexp.MustCompile(`^## \[([^\]]+)\](?:\s*[-—]\s*(.+))?$`)
var changelogH3 = regexp.MustCompile(`^### (.+)$`)

// Changelog parses CHANGELOG.md, cached.
func (c *Context) Changelog() (*Changelog, error) {
	if c.changelog != nil || c.changelogErr != nil {
		return c.changelog, c.changelogErr
	}
	path := filepath.Join(c.RepoRoot, "CHANGELOG.md")
	f, err := os.Open(path)
	if err != nil {
		c.changelogErr = err
		return nil, err
	}
	defer f.Close()

	cl := &Changelog{Path: path}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	sawHeader := false
	var cur *ChangelogSection
	for scanner.Scan() {
		line := scanner.Text()
		if !sawHeader && strings.TrimSpace(line) != "" {
			if strings.TrimSpace(line) == "# Changelog" {
				cl.HasHeader = true
			}
			sawHeader = true
			continue
		}
		if m := changelogVersionLine.FindStringSubmatch(line); m != nil {
			cl.Sections = append(cl.Sections, ChangelogSection{
				Version: m[1],
				Date:    strings.TrimSpace(m[2]),
			})
			cur = &cl.Sections[len(cl.Sections)-1]
			continue
		}
		if cur != nil {
			if m := changelogH3.FindStringSubmatch(line); m != nil {
				cur.Subsections = append(cur.Subsections, strings.TrimSpace(m[1]))
			}
		}
	}
	if err := scanner.Err(); err != nil {
		c.changelogErr = err
		return nil, err
	}
	c.changelog = cl
	return cl, nil
}

// --- Makefile --------------------------------------------------------------

var makefileTargetLine = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_./-]*)\s*:`)

// MakefileTargets returns the set of target names declared in the Makefile.
func (c *Context) MakefileTargets() (map[string]struct{}, error) {
	body, err := c.Makefile()
	if err != nil {
		return nil, err
	}
	out := map[string]struct{}{}
	for _, line := range bytes.Split(body, []byte("\n")) {
		s := string(line)
		// Skip recipe lines (start with TAB) and comments.
		if len(s) > 0 && (s[0] == '\t' || s[0] == '#') {
			continue
		}
		if m := makefileTargetLine.FindStringSubmatch(s); m != nil {
			// Ignore .PHONY etc., which are special targets.
			if strings.HasPrefix(m[1], ".") {
				continue
			}
			out[m[1]] = struct{}{}
		}
	}
	return out, nil
}
