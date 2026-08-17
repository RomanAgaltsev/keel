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

Also give the module an `emits` block naming the checks its workflows report,
the third-party actions they use, and any repository capability they need. See
[the manifest schema](docs/reference/manifest-schema.md#emits). Two guards in
`internal/render/emits_test.go` fail the build if the declaration and the
templates disagree, so an undeclared action or a matrixed check is caught here
rather than on a scaffolded repo.

## Pre-release dogfood

Run this by hand against a throwaway public repo before every minor release.

It exists because the settings module's guarantees are only *partly* checkable
statically. The guards in `internal/render/emits_test.go` cover the three
contradictions that shipped in 2.4.1 — an action the allow-list blocked, a
required check nothing reported, a capability release-please needed and the
settings disabled — but nothing in CI talks to the real API, and
`docs/guides/settings.md`'s design note explains why: applying settings needs an
admin-scoped token, which is more CI risk than it buys.

```bash
keel new --recipe go-library --no-input --answers answers.yaml --target ./demo
gh repo create <owner>/<throwaway> --public --source ./demo --push
export KEEL_GITHUB_TOKEN=...      # needs administration:write
keel settings apply -C ./demo
```

Then push a **`feat:`** commit and assert:

1. every required context reports — `gh pr checks`
2. the PR is `MERGEABLE`, not `BLOCKED` — `gh pr view --json mergeable,mergeStateStatus`
3. the release PR opens

Finally `gh repo delete`.

**The commit type is load-bearing.** #54 survived a green release run because
the scaffold commit was `chore:`: release-please had nothing to release, so it
never touched the endpoint the settings had disabled. The run was green
precisely because it did nothing. A `chore:` commit here proves nothing.

Two things this run is the first real check of, because no test can reach them:

- whether GitHub accepts `docker://rhysd/actionlint` as an `allowed_patterns`
  entry. A Docker action is referenced as `docker://image:tag` rather than
  `owner/repo@version`, so keel allow-lists it bare; if the API rejects that,
  `actionPatterns` in `internal/render/plan.go` is where to fix it.
- whether the aggregate `test` gate job fails rather than skips when a matrix
  cell fails. Break one cell deliberately and confirm the `test` check reports
  `failure`, not `skipped` — a skipped required check does not block, which is
  the defect the gate exists to prevent.
