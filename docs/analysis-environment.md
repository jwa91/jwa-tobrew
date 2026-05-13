# Isolated Analysis Environment

Use this when you want a disposable workspace with the current source
versions of the CLI tools installed together. This is for human/AI analysis,
not release and not convention enforcement.

```sh
scripts/analysis-env.sh
```

The script creates a temporary root with:

```text
src/homebrew-tap/
src/jwa-tobrew/
src/jwa-harden/
src/agentskills/
src/prehandover/
bin/jwa-tobrew
bin/jwa-harden
bin/agentskills
bin/prehandover
```

It clones the selected ref from each repo, builds the Go CLIs from source,
and prints the `PATH` export to use for the session.

Use a fixed root when you want to keep the environment:

```sh
scripts/analysis-env.sh --root /tmp/jwa-analysis
export PATH="/tmp/jwa-analysis/bin:$PATH"
cd /tmp/jwa-analysis/src
```

Use a branch or SHA when you want the analysis environment to reflect a
specific untagged state:

```sh
scripts/analysis-env.sh --root /tmp/jwa-analysis --ref main
```

Use local sibling worktrees when the changes you want to inspect are not
tagged or pushed yet:

```sh
scripts/analysis-env.sh --root /tmp/jwa-analysis --source-root ~/developer
export PATH="/tmp/jwa-analysis/bin:$PATH"
```

This mode locally clones each sibling repo, checks out the same `HEAD`, then
overlays dirty and untracked worktree files into the copy. The result keeps Git
metadata for tag/remote validators while still building and running in the
isolated root. Real `.env` files are not copied; `.env.template` is.

Each repo still owns its own checks. Run repo-local commands from the matching
checkout, for example:

```sh
cd /tmp/jwa-analysis/src/jwa-harden
make check
```
