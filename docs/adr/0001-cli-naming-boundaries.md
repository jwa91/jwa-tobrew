# ADR 0001: CLI Naming And Boundaries

## Status

Proposed before wider release.

## Context

The current family has two adjacent tools:

- `jwa-harden`: runs commands with the repo-local secret and policy envelope.
- `jwa-tobrew`: prepares and publishes projects to the personal Homebrew tap.

The names are serviceable for the private workflow, but they encode history more
than product boundaries. Before broader release, the names should either be
kept intentionally or changed cleanly. Backwards compatibility is not required
before that release.

## Decision

Keep the current binaries for the next private hardening slice, but treat them
as provisional until the first wider release candidate:

- `jwa-harden` owns secret resolution, local policy preflight, and command
  execution through `jwa-harden run -- <command>`.
- `jwa-tobrew` owns release preparation, Homebrew tap publication, scaffold
  generation, and family policy linting.
- Do not add compatibility aliases if the names change before wider release.
  Rename the binaries and update docs/scripts in one clean migration.

## Future Options

- Keep both names if private muscle memory matters more than public clarity.
- Rename `jwa-harden` to a clearer secret/policy runner name.
- Rename `jwa-tobrew` to a release/distribution preparer name.
- Introduce an umbrella `jwa` command only if shared configuration and command
  routing become more valuable than small focused binaries.

## Consequences

Short term, docs must describe the boundary plainly. Long term, the release
candidate should make one explicit naming decision and avoid shims for any
pre-release names that are retired.
