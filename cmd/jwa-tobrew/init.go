package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

func runInit(args []string) error {
	fs := subFlagSet("init", "scaffold release config in the current project")
	kind := fs.String("kind", "", "project kind: go | swift-cask | cask | formula | vps (auto-detected if omitted)")
	name := fs.String("name", "", "tap artifact name (defaults to repo name)")
	desc := fs.String("desc", "", "one-line description for the tap entry")
	force := fs.Bool("force", false, "overwrite existing files")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if !isGitRepo(cwd) {
		return errors.New("current directory is not a git repository")
	}
	owner, repo, err := repoOriginInfo(cwd)
	if err != nil {
		return err
	}
	if *name == "" {
		*name = repo
	}
	if *kind == "" {
		*kind = detectKind(cwd)
	}
	if err := requireAgentskills(); err != nil {
		return err
	}

	switch *kind {
	case "go", "swift-cask", "cask", "formula", "vps":
		return scaffold(*kind, cwd, owner, repo, *name, *desc, *force)
	default:
		return errors.New("could not auto-detect project kind — pass --kind=go|swift-cask|cask|formula|vps")
	}
}

func detectKind(dir string) string {
	if exists(filepath.Join(dir, "go.mod")) {
		return "go"
	}
	for _, pat := range []string{"*.xcodeproj", "*.xcworkspace"} {
		if m, _ := filepath.Glob(filepath.Join(dir, pat)); len(m) > 0 {
			return "swift-cask"
		}
	}
	if exists(filepath.Join(dir, "Package.swift")) {
		return "swift-cask"
	}
	if exists(filepath.Join(dir, "docker-compose.yml")) || exists(filepath.Join(dir, "compose.yml")) {
		return "vps"
	}
	return ""
}

// scaffold writes the release config for a single project kind.
//
// Go projects get a .goreleaser.yaml only — GoReleaser is the entire
// release pipeline (run locally as `goreleaser release` or in a tag-
// triggered GitHub Action), so a shell wrapper would just be one more
// thing to drift. Cask + Formula projects get scripts/release.sh
// because they aren't built by GoReleaser; the script wraps
// `jwa-tobrew release` with the right --kind/--name flags.
func scaffold(kind, dir, owner, repo, name, desc string, force bool) error {
	releaseKind := tapReleaseKind(kind)
	data := map[string]string{
		"Owner": owner,
		"Repo":  repo,
		"Name":  name,
		"Desc":  coalesce(desc, "TODO: one-line description"),
		"Kind":  releaseKind,
	}

	switch kind {
	case "go":
		if err := writeTemplate(filepath.Join(dir, ".goreleaser.yaml"), goreleaserTmpl, data, force); err != nil {
			return err
		}
	case "swift-cask", "cask", "formula":
		releaseShPath := filepath.Join(dir, "scripts", "release.sh")
		if err := writeTemplate(releaseShPath, releaseShTmpl, data, force); err != nil {
			return err
		}
		_ = os.Chmod(releaseShPath, 0o755)
	case "vps":
		// VPS projects still get the shared agent/security contract, but no
		// Homebrew release wrapper. A future backend can own deploy manifests.
	}

	if err := writeAuxiliary(dir, force); err != nil {
		return err
	}

	switch kind {
	case "go":
		ok("scaffolded Go release config (.goreleaser.yaml)")
		hint("cask will be published to %s/homebrew-tap as Casks/%s.rb", owner, name)
		hint("install GoReleaser: brew install goreleaser")
		hint("first release: jwa-harden run -- goreleaser release --clean")
	case "cask":
		ok("scaffolded Cask release config (scripts/release.sh)")
		hint("usage: jwa-harden run -- ./scripts/release.sh <version> <path/to/artifact.dmg>")
	case "swift-cask":
		ok("scaffolded Swift Cask release config (scripts/release.sh)")
		hint("usage: jwa-harden run -- ./scripts/release.sh <version> <path/to/artifact.dmg>")
	case "formula":
		ok("scaffolded Formula release config (scripts/release.sh)")
		hint("usage: jwa-harden run -- ./scripts/release.sh <version> <path/to/binary-or-tarball>")
	case "vps":
		ok("scaffolded VPS agent/security contract")
		hint("release/deploy backend is intentionally not Homebrew-owned yet")
	}
	return nil
}

func tapReleaseKind(kind string) string {
	if kind == "swift-cask" {
		return "cask"
	}
	return kind
}

// writeAuxiliary scaffolds the artifacts needed by every kind: the project's
// .env.template (op:// references for $GITHUB_TOKEN), gitignore protection,
// canonical skills, and a minimal agent contract.
func writeAuxiliary(dir string, force bool) error {
	if err := writeEnvTemplate(filepath.Join(dir, ".env.template"), force); err != nil {
		return err
	}
	if err := ensureGitignoreEntry(filepath.Join(dir, ".gitignore"), ".env"); err != nil {
		return err
	}
	if err := runAgentskills("bootstrap", "--project", dir, "--skill", "release", "--skill", "jwa-harden", "--mode", "copy", "--force"); err != nil {
		return err
	}
	if err := runAgentskills("link", "--project", dir, "--force"); err != nil {
		return err
	}
	if err := writeAgentContracts(dir, force); err != nil {
		return err
	}
	return nil
}

func requireAgentskills() error {
	if _, err := exec.LookPath("agentskills"); err != nil {
		return errors.New("agentskills not found on PATH — install via `brew install jwa91/tap/agentskills`")
	}
	return nil
}

func runAgentskills(args ...string) error {
	cmd := exec.Command("agentskills", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("agentskills %s: %w", strings.Join(args, " "), err)
	}
	return nil
}

func writeEnvTemplate(path string, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			// Don't overwrite — user may have customized the op:// reference.
			return nil
		}
	}
	body := `# Resolved by ` + "`" + `jwa-harden run -- <cmd>` + "`" + ` (delegates to op run).
# Values are 1Password references; resolved secrets exist only in the spawned
# process tree, never on disk. See ~/dotfiles/docs/security-ground-rules.md.
GITHUB_TOKEN=op://Personal/GitHub Homebrew-tap writer/credential
GH_TOKEN=op://Personal/GitHub Homebrew-tap writer/credential
`
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

func writeAgentContracts(dir string, force bool) error {
	rootContract := `# Agent Contract

This repo uses a small shared agent contract:

- Skills live in .agents/skills and harness folders should link there.
- Use jwa-harden run -- <command> for commands that need secrets from .env.template.
- Use jwa-tobrew lint before release-prep changes are considered done.
- Do not commit real .env files; only .env.template with op:// references is allowed.
`
	if err := writeFileIfAllowed(filepath.Join(dir, "AGENTS.md"), []byte(rootContract), force); err != nil {
		return err
	}

	cursorRule := `---
description: Release and secret-handling contract for this repo
alwaysApply: true
---

- Skills live in .agents/skills; harness-specific skill folders should be links.
- Run secret-bearing commands through ` + "`" + `jwa-harden run -- <command>` + "`" + `.
- Run ` + "`" + `jwa-tobrew lint` + "`" + ` after release scaffolding or policy changes.
- Never commit real .env files.
`
	return writeFileIfAllowed(filepath.Join(dir, ".cursor", "rules", "release-contract.mdc"), []byte(cursorRule), force)
}

func writeFileIfAllowed(path string, body []byte, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

func ensureGitignoreEntry(path, entry string) error {
	body, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, line := range strings.Split(string(body), "\n") {
		if strings.TrimSpace(line) == entry {
			return nil
		}
	}
	if len(body) > 0 && !strings.HasSuffix(string(body), "\n") {
		body = append(body, '\n')
	}
	body = append(body, []byte("\n# Forbidden by security ground rules — only .env.template is allowed.\n"+entry+"\n.env.local\n.env.*.local\n")...)
	return os.WriteFile(path, body, 0o644)
}

func writeTemplate(path, tmpl string, data any, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists — pass --force to overwrite", path)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	t, err := template.New(filepath.Base(path)).Parse(tmpl)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, data)
}

func coalesce(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
