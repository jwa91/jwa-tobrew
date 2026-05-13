# jwa-tobrew

`jwa-tobrew` automates publishing a project to the personal Homebrew tap
[`jwa91/homebrew-tap`](https://github.com/jwa91/homebrew-tap) end-to-end:
tag, GitHub Release with auto-generated changelog, asset upload, tap
update + commit + push.

```bash
brew install jwa91/tap/jwa-tobrew
```

It is one of the `jwa-*` family of personal CLIs that share a canonical
command vocabulary (`add`, `bump`, `release`, `init`, `config`, `align`,
`doctor`, `upgrade`).

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
jwa-tobrew doctor              Check tools, tap location, SSH origin, env
jwa-tobrew upgrade             Re-install jwa-tobrew via brew
```

Run `jwa-tobrew <command> -h` for command-specific flags.

## Adding a new project

In the project repo:

```bash
jwa-tobrew doctor       # one-time sanity check
jwa-tobrew init         # auto-detects Go / Cask / Formula
```

`init` requires `agentskills` on PATH. It installs the project `release`
skill from the canonical skill repo and then links the configured agent
harness directories.

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
make check       # go vet + go test
make release VERSION=X.Y.Z   # local release (rare; CI handles tag pushes)
```

## License

MIT
