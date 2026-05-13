# Changelog

All notable changes to this project will be documented in this file.

Format follows [Keep a Changelog](https://keepachangelog.com); versions
follow [SemVer](https://semver.org).

History prior to v0.4.0 was tracked in `jwa91/homebrew-tap` (where
`jwa-tobrew` originally lived as a co-located CLI). The previous release
was v0.3.0 — see [tap CHANGELOG](https://github.com/jwa91/homebrew-tap/blob/main/CHANGELOG.md)
for entries up to and including that version.

## [Unreleased]

## [0.4.2] — 2026-05-13

### Fixed

- **Local release uses `gh auth token` for `GITHUB_TOKEN`** instead of the
  scoped tap-writer PAT, which couldn't create releases on this repo
  (only on `homebrew-tap`). v0.4.1's release attempt failed with HTTP
  403 from the GitHub API. The tap-writer PAT remains in `.env.template`
  as `HOMEBREW_TAP_GITHUB_TOKEN` for the Cask commit step; `GITHUB_TOKEN`
  is now injected by the Makefile from your gh CLI keyring.

## [0.4.1] — 2026-05-13 (never released)

### Fixed

- **macOS Gatekeeper now passes** on first run of `jwa-tobrew` after
  `brew install --cask jwa91/tap/jwa-tobrew`. v0.4.0's cask shipped
  unsigned binaries which Tahoe Gatekeeper blocks with "Apple could not
  verify jwa-tobrew is free of malware". This release codesigns each
  darwin binary with Developer ID + hardened runtime + secure timestamp
  (via `scripts/codesign.sh`, invoked as a goreleaser
  `builds.hooks.post` step) and submits each codesigned binary to
  `xcrun notarytool` (via `scripts/notarize-darwin.sh`, invoked by the
  Makefile release target). The published archive is byte-identical
  pre/post notarization — Apple records the binary's CDHash so the
  Gatekeeper online check passes at install time.

### Changed

- **CI release workflow** (`.github/workflows/release.yml`) is now
  `workflow_dispatch`-only. Tag pushes no longer trigger an automatic
  release because CI does not yet have signing credentials (would
  ship unsigned binaries, which is exactly what 0.4.0 did). The
  canonical release path is `make release VERSION=X.Y.Z` locally,
  with 1Password signed in and the `notarytool` keychain profile set
  up. Adding the Developer ID .p12 + App Store Connect API key to
  GitHub secrets unblocks CI later.
- **`.env.template`** gains `MACOS_SIGN_IDENTITY` (resolves the
  `make-dmg-identity` 1Password item shared with `trnscrb`).
- **Makefile `release`** preflight now also checks for the
  `notarytool` keychain profile and bails early if missing.

## [0.4.0] — 2026-05-13

### Changed

- **Repository extraction**: source moved from
  `jwa91/homebrew-tap/tools/jwa-tobrew/` to this dedicated repo, per
  [ADR 0008 in `homebrew-tap`](https://github.com/jwa91/homebrew-tap/blob/main/docs/adr/0008-one-binary-per-repo-and-homebrew-casks.md).
  The tap is now a pure distribution channel; the publisher (this CLI)
  lives on its own release cadence. Resolves the v0.2.0 namespace
  collision where tap-structural tags and binary tags shared the `v*`
  prefix.
- **Distribution kind**: now ships as a **Homebrew Cask** (via
  GoReleaser's `homebrew_casks:` block) instead of a Formula. End-user
  install is unchanged (`brew install jwa91/tap/jwa-tobrew`); the old
  `Formula/jwa-tobrew.rb` is removed from the tap as part of this cut.
- **Scaffolded `.goreleaser.yaml`** (`jwa-tobrew init --kind=go`) now
  writes a `homebrew_casks:` block with `binaries: [...]` (the modern
  shape) instead of the deprecated `brews:` + singular `binary:`.
- **Source layout**: `tools/jwa-tobrew/*.go` → `cmd/jwa-tobrew/*.go`;
  Go-conventional + matches sibling repos (jwa-harden, prehandover).
- **Two stale `brwtp` legacy references** in error messages
  (`add.go:83`, `tap.go:154`) renamed to `jwa-tobrew`.

### Added

- **Tag-driven release pipeline** in `.github/workflows/release.yml`
  (matching `agentskills` / `jwa-harden` shape). Local releases via
  `make release VERSION=…` still supported.

## [0.3.0] — 2026-05-13 (historical, released from `homebrew-tap`)

See [homebrew-tap CHANGELOG v0.3.0](https://github.com/jwa91/homebrew-tap/blob/main/CHANGELOG.md) for the entry.

[Unreleased]: https://github.com/jwa91/jwa-tobrew/compare/v0.4.2...HEAD
[0.4.2]: https://github.com/jwa91/jwa-tobrew/compare/v0.4.1...v0.4.2
[0.4.1]: https://github.com/jwa91/jwa-tobrew/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/jwa91/jwa-tobrew/releases/tag/v0.4.0
