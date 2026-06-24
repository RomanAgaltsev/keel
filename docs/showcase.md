# What you get

Here's the repository a single `keel new --recipe go-service` run produces —
committed and, if you asked for it, live on your provider. This tree is kept
honest against keel's golden fixture at
`internal/render/testdata/golden/go-service/`.

```text
demo/
├── .github/
│   ├── dependabot.yml
│   └── workflows/
│       ├── actionlint.yml
│       ├── codeql.yml
│       ├── dependency-review.yml
│       ├── govulncheck.yml
│       ├── lint.yml
│       ├── release.yml
│       ├── test.yml
│       └── typos.yml
├── .gitignore
├── .golangci.yml
├── .goreleaser.yaml
├── .release-please-manifest.json
├── .scaffold.lock
├── .typos.toml
├── README.md
├── Taskfile.yml
├── go.mod
├── main.go
└── release-please-config.json
```

## The headline files

- **`Taskfile.yml`** — the task runner: `build`, `lint`, `test`, `cover`, and a
  `ci` gate, with tooling pinned into a project-local `bin/`.
- **`.golangci.yml`** — golangci-lint v2 configuration driving the `lint`
  workflow.
- **`.github/workflows/`** — the full CI surface: `lint`, `test` (race +
  coverage), `codeql` / `govulncheck` / `dependency-review` security scans,
  `actionlint` and `typos` checks, and the `release` pipeline.
- **`.scaffold.lock`** — the record of recipe, modules, versions, answers, and
  per-file hashes that powers [`keel update`](guides/updating.md). See
  [Lockfile](reference/lockfile.md).

Swap the recipe for `rust-service` and the same command yields the Rust
equivalent — a Cargo crate with rustfmt + clippy, a `cargo nextest` matrix,
cargo-audit + cargo-deny scans, and a release-plz + cargo-dist pipeline. See
[Recipes](guides/recipes.md) for the full module breakdown, and the
[Module catalog](reference/module-catalog.md) for what each module contributes.
