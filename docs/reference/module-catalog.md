# Module catalog

The modules built into the keel binary. Most capabilities ship as a matched
pair — one module per language — so a recipe picks the variant that fits its
toolchain. `base-layout` and `spell` are language-agnostic and shared by both
built-in recipes.

| Module | Lang | Description |
|--------|------|-------------|
| `base-layout` | any | README and `.gitignore` common to every repo |
| `spell` | any | Spell-check with crate-ci/typos |
| `go-mod` | go | Go module and `cmd/<name>` entrypoint |
| `cargo-mod` | rust | Minimal Rust crate (`Cargo.toml` + entrypoint) |
| `taskfile-go` / `taskfile-rust` | go / rust | Taskfile with project-local `bin/` tooling and a CI gate |
| `lint-go` | go | golangci-lint v2 config + lint workflow |
| `lint-rust` | rust | rustfmt + clippy config and lint workflow |
| `test-go` | go | race/shuffle test workflow with coverage and optional Codecov upload |
| `test-rust` | rust | `cargo nextest` matrix with optional Codecov coverage |
| `security-go` | go | One `security.yml`: CodeQL, govulncheck, dependency-review, workflow linting |
| `security-rust` | rust | cargo-audit + cargo-deny, dependency-review, workflow linting |
| `dep-bots-go` / `dep-bots-rust` | go / rust | Dependabot or Renovate dependency-update config |
| `release-go` | go | release-please + GoReleaser release pipeline with a conventional-commit gate |
| `release-rust` | rust | release-plz + cargo-dist release pipeline with a conventional-commit gate |
| `license` | any | `LICENSE` for the chosen license |
| `governance` | any | `SECURITY.md`, `.github/CODEOWNERS`, `.editorconfig` |
| `contributing-go` / `contributing-rust` | go / rust | `CONTRIBUTING.md` for the language |
| `community-templates` | any | Issue forms and pull-request template |

`keel list` prints the live set for your installed version — treat it as
authoritative. Each module's own questions are documented by example in
[Authoring → Modules](../authoring/modules.md), and the recipes that compose
these modules are in [Guides → Recipes](../guides/recipes.md).
