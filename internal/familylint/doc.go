// Package familylint defines the executable contract every jwa-* CLI repo
// must satisfy. Rules self-register from init() functions into a global
// registry; consumers iterate the registry and invoke each rule's Check
// against a Context bound to a particular repo (and optionally a built
// binary).
//
// Rule IDs follow the pattern F-<layer>-<NNN>:
//
//	F-cmd-*    binary command surface (requires the binary to run)
//	F-io-*     exit-code / stdout-stderr conventions
//	F-repo-*   repository file structure
//	F-cfg-*    config-file contents (.goreleaser.yaml, Makefile,
//	           CHANGELOG.md, .github/workflows/release.yml)
//	F-ver-*    version / tag / release identity coherence
//
// No CLI subcommand consumes this package yet. The library is the
// validator surface; a caller (future jwa-tobrew lint, prek hook, etc.)
// wires it to its execution context.
package familylint
