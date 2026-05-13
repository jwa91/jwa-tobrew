# jwa-tobrew

`jwa-tobrew` automates publishing a project to the personal Homebrew tap
[`jwa91/homebrew-tap`](https://github.com/jwa91/homebrew-tap) end-to-end:
tag, GitHub Release with auto-generated changelog, asset upload, tap
update + commit + push.

```bash
brew install --cask jwa91/tap/jwa-tobrew
```

See [Installation](docs/installation.md) for Homebrew, direct download,
`go install`, and source-build options.

For disposable AI/human inspection with untagged source builds of the CLI
tools, see [Isolated Analysis Environment](docs/analysis-environment.md).

It is one of the `jwa-*` family of personal CLIs that share a canonical
command vocabulary (`add`, `bump`, `release`, `init`, `config`, `align`,
`completion`, `doctor`, `lint`, `upgrade`, `version`).

## Secrets

`jwa-tobrew` never touches 1Password directly. It reads `$GITHUB_TOKEN`
from its environment and errors out if the env is empty when the GitHub
API is needed. Wrap with
[`jwa-harden`](https://github.com/jwa91/jwa-harden):

```bash
jwa-harden run -- jwa-tobrew release --kind go --version 0.2.0
```

See [`~/dotfiles/docs/security-ground-rules.md`](https://github.com/jwa91/dotfiles)
for the full security model and
[ADR 0001](https://github.com/jwa91/homebrew-tap/blob/main/docs/adr/0001-no-secret-handling-in-jwa-tobrew.md)
for why this binary is secret-blind by design.

## Commands

```
jwa-tobrew add <github-url>    Snapshot a published GitHub release into the tap (no token)
jwa-tobrew bump <name> [ver]   Re-sync an existing tap entry to a published release
jwa-tobrew release             Tag + release a project + update the tap (run inside the project)
jwa-tobrew init                Scaffold release config in another project
jwa-tobrew config              Regenerate tap.toml + tap.local.toml + README items table
jwa-tobrew deps                Show dependency overview for every item in the tap
jwa-tobrew align               Detect drift between repo state and tap conventions
jwa-tobrew completion <shell>  Generate shell completion for bash, zsh, or fish
jwa-tobrew doctor              Check tools, tap location, SSH origin, env
jwa-tobrew lint                Run the jwa-* family policy lint rules
jwa-tobrew upgrade             Re-install jwa-tobrew via brew
jwa-tobrew version             Print build info
```

Run `jwa-tobrew <command> -h` for command-specific flags.

Install shell completion by writing the generated script to your shell's
completion directory:

```bash
jwa-tobrew completion zsh > ~/.zfunc/_jwa-tobrew
jwa-tobrew completion bash > ~/.local/share/bash-completion/completions/jwa-tobrew
jwa-tobrew completion fish > ~/.config/fish/completions/jwa-tobrew.fish
```

Exit code contract: `0` success/help, `1` runtime failure, `2` usage error,
`4` policy/lint failure. Human output is the default; policy checks also expose
`--format=json` for agents.

### Policy Lint Repo Kinds

`jwa-tobrew lint` selects a policy pack with `--kind` or by repo shape:

- `go-cli`: Go command-line tools released through GoReleaser and
  `homebrew_casks`.
- `swift-cask`: Swift/macOS apps released as Homebrew Casks with a
  `scripts/release.sh` wrapper.
- `cask`: prebuilt app or binary artifacts released as Homebrew Casks.
- `formula`: single binary or archive artifacts released as Homebrew Formulae.
- `generic`: shared repo hygiene for projects that are not release-shaped yet.
- `vps`: deployment projects that share agent/security contracts but do not
  have a Homebrew release backend yet.

Go-only release rules are intentionally scoped to `go-cli`. Other kinds receive
the shared command, IO, repository hygiene, changelog, and tag checks without
inheriting GoReleaser-specific requirements.

## Adding a new project

In the project repo:

```bash
jwa-tobrew doctor       # one-time sanity check
jwa-tobrew init         # auto-detects Go / Cask / Formula
```

`init` requires `agentskills` on PATH. It installs the project `release`
skill from the canonical skill repo and then links the configured agent
harness directories. It also writes a minimal agent contract (`AGENTS.md` and
Cursor rules) so future agents know where skills live and how releases run.

Then release. The flow differs by kind on purpose — Go projects use
GoReleaser as the entire pipeline (same command locally and in CI),
while casks and formulas use a thin `scripts/release.sh` wrapper around
`jwa-tobrew release`:

```bash
# Go (after `init --kind=go`):
jwa-harden run -- goreleaser release --clean

# Cask / Formula (after `init --kind=cask` or `init --kind=formula`):
jwa-harden run -- ./scripts/release.sh 0.1.0 path/to/artifact
```

For Go projects, GoReleaser handles cross-platform builds, the
changelog, and auto-commits a Cask to the tap (modern shape per
[ADR 0008](https://github.com/jwa91/homebrew-tap/blob/main/docs/adr/0008-one-binary-per-repo-and-homebrew-casks.md)
in the tap repo). For Casks (macOS apps) and other binaries,
`jwa-tobrew` handles tag → GitHub release → asset upload → tap update
directly.

### Snapshotting an existing release (no source changes needed)

```bash
jwa-tobrew add github.com/SOMEONE/some-cli                   # latest release, auto-detect asset
jwa-tobrew add github.com/SOMEONE/some-cli --version 1.2.3
jwa-tobrew add github.com/SOMEONE/some-app --kind cask --asset SomeApp.dmg
```

This requires no token at all — push to the tap is plain SSH.

## Development

```bash
make build       # build to ./bin/jwa-tobrew
make install     # build + install to ~/.local/bin/jwa-tobrew
make check       # go vet + family lint + go test
make release VERSION=X.Y.Z   # local release (rare; CI handles tag pushes)
```

## License

MIT
