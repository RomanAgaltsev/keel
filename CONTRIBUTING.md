# Contributing to keel

## Module versioning

Every module under `modules/<name>/` carries a semver `version:` in its
`module.yaml`. It is the signal `keel update` (and `keel outdated`) reads, so any
change to a module's files MUST bump its version:

- **patch** (`x.y.Z`) — safe-to-re-apply content change: action/tool/SHA bump,
  typo, comment, formatting. No files added/removed; no question-schema change.
- **minor** (`x.Y.0`) — backward-compatible addition: a new template file, a new
  *optional* question or file rule.
- **major** (`X.0.0`) — breaking: removed/renamed template file, removed/renamed/
  retyped question id, a newly *required* question, a changed `dest`.

CI enforces this (`module-version` workflow / `task modules:check`). Renovate
PRs auto-bump (patch) via `task modules:bump`; for manual changes, run
`go run ./internal/tools/modulebump <module> <level>` and commit the result.

## Adding a module that ships workflows

If a new module's `templates/` tree contains `.github/workflows/`, add its
templates directory to the `github-actions` block in `.github/dependabot.yml`
that lists `/modules/*/templates`.

dependabot's `github-actions` ecosystem only scans the directories it is told
about. The entry at `directory: /` sees this repo's own workflows and nothing
else, which is why an emitted template sat on `actions/checkout@v5` for months
while keel's own workflows moved to `@v7`. A module whose templates are not in
that list gets no dependency updates, and its pins go stale silently.

`internal/render/pins_test.go` fails the build on a known-stale pin, but it can
only catch the ones it knows about — the dependabot entry is what keeps the
templates current going forward.
