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
