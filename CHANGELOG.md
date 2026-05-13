# Changelog

All notable changes to this project will be documented in this file.

Format follows [Keep a Changelog](https://keepachangelog.com); versions
follow [SemVer](https://semver.org).

History prior to v0.4.0 was tracked in `jwa91/homebrew-tap` (where
`jwa-tobrew` originally lived as a co-located CLI). The previous release
was v0.3.0 — see [tap CHANGELOG](https://github.com/jwa91/homebrew-tap/blob/main/CHANGELOG.md)
for entries up to and including that version.

## [Unreleased]

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

[Unreleased]: https://github.com/jwa91/jwa-tobrew/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/jwa91/jwa-tobrew/releases/tag/v0.4.0
