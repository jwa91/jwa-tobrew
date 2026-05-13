# Policy Packs

`jwa-tobrew lint` groups family rules by repository kind. Use
`--kind=auto` for shape detection or pass a kind explicitly when validating a
scaffold that does not yet have enough files to detect.

## Kinds

- `go-cli`: Go command-line tools distributed as Homebrew Casks through
  GoReleaser `homebrew_casks`.
- `swift-cask`: Swift or Xcode-based macOS apps distributed as Homebrew Casks.
- `cask`: non-Swift prebuilt app or binary artifacts distributed as Homebrew
  Casks.
- `formula`: binary or archive artifacts distributed as Homebrew Formulae.
- `generic`: shared family hygiene before a release backend is selected.
- `vps`: server/deployment repos that share agent and secret-handling contracts.

## Rule Packs

Every kind gets shared command and IO checks when a binary is supplied, plus
shared repository hygiene, changelog, and tag checks. The `go-cli` pack adds
GoReleaser, Makefile release, version-output, codesigning, notarization, and
Homebrew Cask publication checks.

Swift, Cask, Formula, Generic, and VPS packs deliberately do not inherit
Go-only GoReleaser assumptions. Their release/deploy-specific rules should be
added as new, explicitly named pack sections instead of widening the `go-cli`
pack.
