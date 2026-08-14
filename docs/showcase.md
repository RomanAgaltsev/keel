# What you get

Here's the repository a single `keel new --recipe go-service` run produces —
committed and, if you asked for it, live on your provider. This tree is kept
honest against keel's golden fixture at
`internal/render/testdata/golden/go-service/`.

```text
demo/
├── .github/
│   ├── ISSUE_TEMPLATE/
│   │   ├── bug_report.yml
│   │   ├── config.yml
│   │   └── feature_request.yml
│   ├── codeql/
│   │   └── codeql-config.yml
│   ├── workflows/
│   │   ├── lint.yml
│   │   ├── pr-title.yml
│   │   ├── release.yml
│   │   ├── security.yml
│   │   ├── test.yml
│   │   └── typos.yml
│   ├── CODEOWNERS
│   ├── PULL_REQUEST_TEMPLATE.md
│   └── dependabot.yml
├── cmd/
│   └── demo/
│       └── main.go
├── .editorconfig
├── .gitignore
├── .golangci.yml
├── .goreleaser.yaml
├── .release-please-manifest.json
├── .scaffold.lock
├── .typos.toml
├── CONTRIBUTING.md
├── LICENSE
├── README.md
├── SECURITY.md
├── Taskfile.yml
├── codecov.yml
├── go.mod
└── release-please-config.json
```

`CODEOWNERS` appears when you answer the `code_owner` question; `codecov.yml` and
the Codecov upload step when you enable `enable_codecov`; `codeql-config.yml` when
you enable CodeQL.

## The headline files

- **`Taskfile.yml`** — the task runner: `build`, `lint`, `test`, `cover`, and a
  `ci` gate, with tooling pinned into a project-local `bin/`.
- **`.golangci.yml`** — golangci-lint v2 configuration driving the `lint`
  workflow.
- **`.github/workflows/`** — the full CI surface: `lint`, `test` (race +
  coverage), one `security.yml` carrying the `codeql` / `govulncheck` /
  `dependency-review` / `actionlint` jobs, the `typos` check, the `pr-title`
  conventional-commit gate, and the `release` pipeline.
- **`cmd/<name>/main.go`** — the entrypoint, with `version` / `commit` / `date`
  vars the Taskfile and GoReleaser inject through `-ldflags`.
- **`LICENSE`, `CONTRIBUTING.md`, `SECURITY.md`, `CODEOWNERS`, issue forms and a
  PR template** — the governance surface a public repo needs, rendered from your
  answers instead of copied by hand.
- **`.scaffold.lock`** — the record of recipe, modules, versions, answers, and
  per-file hashes that powers [`keel update`](guides/updating.md). See
  [Lockfile](reference/lockfile.md).

Swap the recipe for `rust-service` and the same command yields the Rust
equivalent — a Cargo crate with rustfmt + clippy, a `cargo nextest` matrix,
cargo-audit + cargo-deny scans, and a release-plz + cargo-dist pipeline. See
[Recipes](guides/recipes.md) for the full module breakdown, and the
[Module catalog](reference/module-catalog.md) for what each module contributes.
