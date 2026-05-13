package familylint

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Context is the per-repo state a Rule.Check inspects. Construct with
// NewContext + Options. All getters are lazy + cached; calling the same
// getter twice is cheap.
type Context struct {
	// RepoRoot is the absolute path to the jwa-* CLI repo under inspection.
	RepoRoot string
	// RepoName defaults to filepath.Base(RepoRoot). Override via WithRepoName.
	RepoName string
	// BinaryPath is an absolute path to a built binary. Empty when not set;
	// rules in LayerCmd Skip rather than fail.
	BinaryPath string
	// RepoKind selects the active familylint rule pack.
	RepoKind RepoKind
	// DotfilesBrewfile points to the machine-level Brewfile when available.
	DotfilesBrewfile string

	// Cached parsed artefacts. nil + nil err means "not yet read".
	makefile      []byte
	makefileErr   error
	readme        []byte
	readmeErr     error
	gitignore     []byte
	gitignoreErr  error
	workflow      []byte
	workflowErr   error
	goreleaser    *Goreleaser
	goreleaserErr error
	changelog     *Changelog
	changelogErr  error
	gitTags       []string
	gitTagsErr    error
}

// Option configures a Context at construction time.
type Option func(*Context)

// WithBinary sets the path to a built binary so LayerCmd rules can
// execute it. Resolved to an absolute path internally.
func WithBinary(path string) Option {
	return func(c *Context) {
		if path == "" {
			return
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			c.BinaryPath = path // best effort; rules will report exec errors
			return
		}
		c.BinaryPath = abs
	}
}

// WithRepoName overrides the RepoName (defaults to the directory base).
// Use when the on-disk path doesn't match the canonical repo name.
func WithRepoName(name string) Option {
	return func(c *Context) { c.RepoName = name }
}

// WithRepoKind overrides automatic repo-kind detection.
func WithRepoKind(kind RepoKind) Option {
	return func(c *Context) { c.RepoKind = kind }
}

// WithDotfilesBrewfile overrides the machine Brewfile path used by family-order checks.
func WithDotfilesBrewfile(path string) Option {
	return func(c *Context) { c.DotfilesBrewfile = path }
}

// NewContext builds a Context rooted at repoRoot. The path must exist
// as a directory. Apply zero or more Options to configure further.
func NewContext(repoRoot string, opts ...Option) (*Context, error) {
	abs, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("abs %s: %w", repoRoot, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", abs, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("repo root is not a directory: %s", abs)
	}
	c := &Context{
		RepoRoot:         abs,
		RepoName:         filepath.Base(abs),
		RepoKind:         detectRepoKind(abs),
		DotfilesBrewfile: defaultDotfilesBrewfile(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

func detectRepoKind(root string) RepoKind {
	switch {
	case fileExists(filepath.Join(root, "go.mod")):
		return RepoKindGoCLI
	case globExists(filepath.Join(root, "*.xcodeproj")), globExists(filepath.Join(root, "*.xcworkspace")), fileExists(filepath.Join(root, "Package.swift")):
		return RepoKindSwiftCask
	case fileExists(filepath.Join(root, "scripts", "release.sh")):
		return RepoKindCask
	case fileExists(filepath.Join(root, "docker-compose.yml")), fileExists(filepath.Join(root, "compose.yml")):
		return RepoKindVPS
	default:
		return RepoKindGeneric
	}
}

func defaultDotfilesBrewfile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "dotfiles", "Brewfile")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func globExists(pattern string) bool {
	matches, err := filepath.Glob(pattern)
	return err == nil && len(matches) > 0
}

// FileExists reports whether a path relative to RepoRoot exists.
func (c *Context) FileExists(rel string) bool {
	_, err := os.Stat(filepath.Join(c.RepoRoot, rel))
	return err == nil
}

// FileExecutable reports whether rel exists and has any execute bit set.
func (c *Context) FileExecutable(rel string) bool {
	info, err := os.Stat(filepath.Join(c.RepoRoot, rel))
	if err != nil {
		return false
	}
	return info.Mode()&0o111 != 0
}

// ReadFile returns rel's contents.
func (c *Context) ReadFile(rel string) ([]byte, error) {
	return os.ReadFile(filepath.Join(c.RepoRoot, rel))
}

// readOnceFn captures the lazy-load pattern for cached file accessors.
// Returns the cached bytes after the first successful or failing read.
func (c *Context) readOnceFn(cached *[]byte, cachedErr *error, read func() ([]byte, error)) ([]byte, error) {
	if *cached != nil {
		return *cached, nil
	}
	if *cachedErr != nil {
		return nil, *cachedErr
	}
	body, err := read()
	*cached = body
	*cachedErr = err
	return body, err
}

// Makefile returns Makefile bytes, cached. os.ErrNotExist when absent.
func (c *Context) Makefile() ([]byte, error) {
	return c.readOnceFn(&c.makefile, &c.makefileErr, func() ([]byte, error) {
		return c.ReadFile("Makefile")
	})
}

// README returns README.md bytes, cached.
func (c *Context) README() ([]byte, error) {
	return c.readOnceFn(&c.readme, &c.readmeErr, func() ([]byte, error) {
		return c.ReadFile("README.md")
	})
}

// Gitignore returns .gitignore bytes, cached.
func (c *Context) Gitignore() ([]byte, error) {
	return c.readOnceFn(&c.gitignore, &c.gitignoreErr, func() ([]byte, error) {
		return c.ReadFile(".gitignore")
	})
}

// ReleaseWorkflow returns .github/workflows/release.yml bytes, cached.
func (c *Context) ReleaseWorkflow() ([]byte, error) {
	return c.readOnceFn(&c.workflow, &c.workflowErr, func() ([]byte, error) {
		return c.ReadFile(".github/workflows/release.yml")
	})
}

// CmdOut captures one binary invocation.
type CmdOut struct {
	Stdout  string
	Stderr  string
	Exit    int   // -1 when ExecErr is non-nil
	ExecErr error // start/signal failures; non-zero exit is in Exit, not here
}

// errNoBinary signals a Context with no configured BinaryPath.
var errNoBinary = errors.New("no binary path configured (apply WithBinary)")

// RunBinary invokes BinaryPath with args, capturing stdout/stderr/exit.
// Returns ExecErr=errNoBinary when BinaryPath wasn't set. Sets NO_COLOR=1
// in the child env so ANSI escapes don't disturb pattern checks.
func (c *Context) RunBinary(args ...string) CmdOut {
	if c.BinaryPath == "" {
		return CmdOut{Exit: -1, ExecErr: errNoBinary}
	}
	cmd := exec.Command(c.BinaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	err := cmd.Run()
	out := CmdOut{Stdout: stdout.String(), Stderr: stderr.String()}
	if err == nil {
		return out
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		out.Exit = exitErr.ExitCode()
		return out
	}
	out.Exit = -1
	out.ExecErr = err
	return out
}

// GitOrigin returns the URL of the `origin` remote.
func (c *Context) GitOrigin() (string, error) {
	cmd := exec.Command("git", "-C", c.RepoRoot, "remote", "get-url", "origin")
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git remote get-url origin: %w (%s)", err, strings.TrimSpace(errOut.String()))
	}
	return strings.TrimSpace(out.String()), nil
}

// GitTags returns v* tags reachable from HEAD in most-recent-first order, cached.
func (c *Context) GitTags() ([]string, error) {
	if c.gitTags != nil || c.gitTagsErr != nil {
		return c.gitTags, c.gitTagsErr
	}
	cmd := exec.Command("git", "-C", c.RepoRoot, "tag", "--list", "--sort=-creatordate", "v*")
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		c.gitTagsErr = fmt.Errorf("git tag --list: %w (%s)", err, strings.TrimSpace(errOut.String()))
		return nil, c.gitTagsErr
	}
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		c.gitTags = append(c.gitTags, line)
	}
	return c.gitTags, nil
}

// LatestTag returns the most recent v* tag, or "" if none exist.
func (c *Context) LatestTag() (string, error) {
	tags, err := c.GitTags()
	if err != nil {
		return "", err
	}
	if len(tags) == 0 {
		return "", nil
	}
	return tags[0], nil
}

// WalkRepo iterates files under RepoRoot, skipping common build/VCS noise.
func (c *Context) WalkRepo(fn func(rel string, info fs.FileInfo) error) error {
	return filepath.Walk(c.RepoRoot, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(c.RepoRoot, path)
		if relErr != nil {
			return relErr
		}
		if info.IsDir() {
			switch info.Name() {
			case ".git", "node_modules", "dist", "bin", "vendor", ".venv", "__pycache__":
				return filepath.SkipDir
			}
			return nil
		}
		return fn(rel, info)
	})
}
