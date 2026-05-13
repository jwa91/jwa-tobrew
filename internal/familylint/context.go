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
// NewContext. All file/process helpers are lazy + cached where it makes
// sense; calling the same getter twice is cheap.
type Context struct {
	// RepoRoot is the absolute path to the jwa-* CLI repo under inspection.
	RepoRoot string
	// RepoName is filepath.Base(RepoRoot) by default. Override if the
	// repo's canonical name differs from its directory name (rare).
	RepoName string
	// BinaryPath is an absolute path to a built binary for this repo. When
	// empty, rules in LayerCmd Skip rather than fail.
	BinaryPath string

	// Cached parsed artefacts. nil + nil err means "not yet read".
	makefile     []byte
	makefileErr  error
	readme       []byte
	readmeErr    error
	gitignore    []byte
	gitignoreErr error
	workflow     []byte
	workflowErr  error
	goreleaser   *Goreleaser
	goreleaserErr error
	changelog    *Changelog
	changelogErr error
	gitTags      []string
	gitTagsErr   error
}

// NewContext builds a Context rooted at repoRoot. The path must exist as
// a directory. binaryPath may be empty.
func NewContext(repoRoot, binaryPath string) (*Context, error) {
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
	binAbs := ""
	if binaryPath != "" {
		binAbs, err = filepath.Abs(binaryPath)
		if err != nil {
			return nil, fmt.Errorf("abs %s: %w", binaryPath, err)
		}
	}
	return &Context{
		RepoRoot:   abs,
		RepoName:   filepath.Base(abs),
		BinaryPath: binAbs,
	}, nil
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

func cacheFile(path string, cached *[]byte, cachedErr *error, read func() ([]byte, error)) ([]byte, error) {
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

// Makefile returns the raw Makefile bytes, cached. os.ErrNotExist when absent.
func (c *Context) Makefile() ([]byte, error) {
	return cacheFile("Makefile", &c.makefile, &c.makefileErr, func() ([]byte, error) {
		return c.ReadFile("Makefile")
	})
}

// README returns README.md bytes, cached.
func (c *Context) README() ([]byte, error) {
	return cacheFile("README.md", &c.readme, &c.readmeErr, func() ([]byte, error) {
		return c.ReadFile("README.md")
	})
}

// Gitignore returns .gitignore bytes, cached.
func (c *Context) Gitignore() ([]byte, error) {
	return cacheFile(".gitignore", &c.gitignore, &c.gitignoreErr, func() ([]byte, error) {
		return c.ReadFile(".gitignore")
	})
}

// ReleaseWorkflow returns .github/workflows/release.yml bytes, cached.
func (c *Context) ReleaseWorkflow() ([]byte, error) {
	return cacheFile("release.yml", &c.workflow, &c.workflowErr, func() ([]byte, error) {
		return c.ReadFile(".github/workflows/release.yml")
	})
}

// CmdOut captures one binary invocation.
type CmdOut struct {
	Stdout string
	Stderr string
	// Exit is the process exit code. -1 when ExecErr is non-nil (process
	// didn't start cleanly — binary missing, signal, etc.).
	Exit    int
	ExecErr error
}

// RunBinary invokes BinaryPath with args, capturing stdout/stderr/exit.
// Returns CmdOut{ExecErr: errNoBinary} when BinaryPath is empty. Sets
// NO_COLOR=1 in the child env so ANSI escape sequences don't confuse rule
// pattern checks.
func (c *Context) RunBinary(args ...string) CmdOut {
	if c.BinaryPath == "" {
		return CmdOut{Exit: -1, ExecErr: errors.New("no binary path configured")}
	}
	cmd := exec.Command(c.BinaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	err := cmd.Run()
	out := CmdOut{Stdout: stdout.String(), Stderr: stderr.String()}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			out.Exit = exitErr.ExitCode()
		} else {
			out.Exit = -1
			out.ExecErr = err
		}
	}
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

// GitTags returns annotated/lightweight tags matching v* in
// most-recent-first creator-date order, cached.
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
