# keel

[![ci](https://github.com/RomanAgaltsev/keel/actions/workflows/ci.yml/badge.svg)](https://github.com/RomanAgaltsev/keel/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/RomanAgaltsev/keel/branch/main/graph/badge.svg)](https://codecov.io/gh/RomanAgaltsev/keel)
[![Go Reference](https://pkg.go.dev/badge/github.com/RomanAgaltsev/keel.svg)](https://pkg.go.dev/github.com/RomanAgaltsev/keel)
[![Go Report Card](https://goreportcard.com/badge/github.com/RomanAgaltsev/keel)](https://goreportcard.com/report/github.com/RomanAgaltsev/keel)
[![Release](https://img.shields.io/github/v/release/RomanAgaltsev/keel)](https://github.com/RomanAgaltsev/keel/releases)
[![License: MIT](https://img.shields.io/github/license/RomanAgaltsev/keel)](./LICENSE)

**Scaffold a new git repository from composable template modules.** 

`keel` turns a *recipe* — a named list of small, single-purpose modules — into a ready-to-push repository: the source layout, the Taskfile, the CI workflows, the linter and
release config, all rendered from your answers. It then initializes git, writes
the first commit, and (optionally) creates the remote on GitHub and pushes —
in one command.

```bash
keel new --recipe go-service
```

That one command, answered interactively, produces a Go service with a module
layout, a `Taskfile`, golangci-lint v2, a race/coverage test workflow, CodeQL +
govulncheck + dependency-review security scans, Dependabot/Renovate, a
release-please + GoReleaser pipeline, and a typos spell-check — committed and, if
you asked for it, live on GitHub.

Swap the recipe for `rust-service` and the same command produces a Cargo crate
with the equivalent Rust toolchain — rustfmt + clippy, a `cargo nextest` matrix,
cargo-audit + cargo-deny security scans, and a release-plz + cargo-dist pipeline:

```bash
keel new --recipe rust-service
```

## Why keel?

- **Composition over monoliths.** A repository is assembled from independent
  modules (`base-layout`, `go-mod`, `lint-go`, `release-rust`, …). Add a
  capability — or a whole new language — by adding a module to a recipe, not by
  editing a giant template.
- **State-aware, not just a file dump.** `keel` detects whether the target
  directory and the remote already exist, and branches accordingly: fresh write,
  overlay onto an existing tree, clone-then-overlay, or hand reconciliation back
  to you when both sides already have history. It never force-pushes over your
  work.
- **Idempotent and reproducible.** Every run records a `.scaffold.lock` (keel
  version, recipe, modules and their versions, and the answers). Re-running with
  the same answers produces no diff and no empty commit.
- **Interactive or hands-off.** Answer a typed wizard, supply an answers file, or
  run `--no-input` for CI. A `--dry-run` prints the plan without touching disk or
  network.
- **Dependency-light, self-contained.** Modules and recipes are embedded into the
  binary (`go:embed`), so a single `keel` executable carries everything it needs.

## Install

```bash
go install github.com/RomanAgaltsev/keel/cmd/keel@latest
```

Or grab a binary from the [releases page](https://github.com/RomanAgaltsev/keel/releases).

Requires Go 1.26+ to build from source.

## Quick start

```bash
# Interactive: keel asks for repo name, module path, author, license, etc.
keel new --recipe go-service

# See what would happen without writing anything or hitting the network.
keel new --recipe go-service --dry-run

# Non-interactive (CI): take every answer from a file, never prompt.
keel new --recipe go-service --answers answers.yaml --no-input

# Scaffold a Rust crate instead.
keel new --recipe rust-service
```

Ready-to-run answers files, a custom recipe, and an example external module live
in [`examples/`](examples/).

An answers file mirrors the question IDs:

```yaml
# answers.yaml
repo_name: demo
description: a demo service
module_path: github.com/you/demo
author_name: Your Name
author_email: you@example.com
license: MIT
visibility: public
provider: github
create_remote: true
```

## Commands

| Command | What it does |
|---------|--------------|
| `keel new` | Scaffold a repository from a recipe. |
| `keel list` | List the available recipes and modules. |
| `keel config` | Manage keel's user config (`get` / `set` / `list`). |
| `keel version` | Print version, commit, and build date. |
| `keel outdated` | Report which of a repo's keel modules have newer versions available. |
| `keel update` | Re-apply evolved module templates to an existing repo (hash-aware overlay; user edits are preserved as `.keel-new`). |


### `keel new` flags

| Flag | Default | Effect |
|------|---------|--------|
| `--recipe` | `go-service` | Which recipe to scaffold. |
| `--target` | repo name | Target directory. |
| `--answers` | — | Read answers from a YAML file. |
| `--no-input` | `false` | Never prompt; fail if a required answer is missing (CI mode). |
| `--remote-url` | — | Wire/clone an existing remote instead of creating one. |
| `--overwrite` | `false` | Overwrite existing files when overlaying onto a tree. |
| `--dry-run` | `false` | Print the plan; touch neither disk nor network. |

### Updating a repo

`keel update` re-renders a repo's modules at their current template versions and
overlays the result, using the per-file hashes recorded in `.scaffold.lock`:

- An untouched file is updated in place.
- A file you edited is left alone; the new render is written alongside as
  `<path>.keel-new` for you to merge. `--overwrite` replaces it instead.
- `--dry-run` previews the classification; `--reconfigure` re-asks the questions;
  `--commit` makes a single `chore: keel update` commit of keel's own changes.

`keel outdated` reports, without changing anything, which modules have a newer
version available.

## How it works

A `keel new` run is a small state machine:

1. **Resolve the recipe** into its ordered list of modules and load their
   manifests.
2. **Collect answers** — merge the built-in core questions (repo name, module
   path, author, license, visibility, provider) with each module's own questions,
   then fill them from the answers file and/or the interactive wizard.
3. **Build a render plan** — expand every module's `files` rules (templated with
   Go `text/template`, gated by optional `when` conditions) into a single set of
   destination files. Cross-module collisions fail the plan early.
4. **Detect state** — is the target directory present? does the remote exist?
5. **Materialize** the plan, branching on state:
   - target absent → fresh atomic write
   - target present → overlay (skip or `--overwrite`)
   - local absent, remote present → clone-then-overlay
6. **Commit** — `git init` (branch `main`), set author identity, write
   `.scaffold.lock`, stage, and commit `chore: scaffold with keel` (skipped when
   nothing changed).
7. **Remote** — if requested, create the GitHub repo (when it doesn't already
   exist), wire `origin`, and push. When both local *and* remote already have
   history, keel refuses to force anything and prints the fetch/rebase/push steps
   for you to run.

### Modules

A module is a directory with a `module.yaml` manifest and a `templates/` tree:

```yaml
# modules/security-go/module.yaml
name: security-go
description: CodeQL, govulncheck, dependency-review, workflow linting
version: 1.0.0
language: go
requires: [base-layout]
questions:
  - id: enable_codeql
    prompt: "Enable CodeQL scanning?"
    type: bool
    default: true
files:
  - src: ".github/workflows/codeql.yml"
    dest: "."
    when: "{{ .enable_codeql }}"     # optional condition gating the file
  - src: ".github/workflows/dependency-review.yml"
    dest: "."
```

Each module contributes its own questions and its own files; `requires` declares
dependencies so a recipe stays consistent. The `language` field (`go`, `rust`, or
`any`) keeps a recipe's modules pinned to a single toolchain.

### Recipes

A recipe is just a named composition of modules:

```yaml
# recipes/go-service.yaml
name: go-service
language: go
modules: [base-layout, go-mod, taskfile-go, lint-go, test-go, security-go, dep-bots-go, release-go, spell]
```

```yaml
# recipes/rust-service.yaml
name: rust-service
language: rust
modules: [base-layout, cargo-mod, taskfile-rust, lint-rust, test-rust, security-rust, release-rust, dep-bots-rust, spell]
```

## Built-in modules

Most capabilities ship as a matched pair — one module per language — so a recipe
picks the variant that fits its toolchain. `base-layout` and `spell` are
language-agnostic and shared by both recipes.

| Module | Lang | Description |
|--------|------|-------------|
| `base-layout` | any | README and `.gitignore` common to every repo |
| `spell` | any | Spell-check with crate-ci/typos |
| `go-mod` | go | Minimal Go module and entrypoint |
| `cargo-mod` | rust | Minimal Rust crate (`Cargo.toml` + entrypoint) |
| `taskfile-go` / `taskfile-rust` | go / rust | Taskfile with project-local `bin/` tooling and a CI gate |
| `lint-go` | go | golangci-lint v2 config + lint workflow |
| `lint-rust` | rust | rustfmt + clippy config and lint workflow |
| `test-go` | go | race/shuffle test workflow with coverage |
| `test-rust` | rust | `cargo nextest` matrix with optional Codecov coverage |
| `security-go` | go | CodeQL, govulncheck, dependency-review, workflow linting |
| `security-rust` | rust | cargo-audit + cargo-deny, dependency-review, workflow linting |
| `dep-bots-go` / `dep-bots-rust` | go / rust | Dependabot or Renovate dependency-update config |
| `release-go` | go | release-please + GoReleaser release pipeline |
| `release-rust` | rust | release-plz (version/changelog) + cargo-dist (binaries) |

Two recipes compose these into production-ready services:

- **`go-service`** — the full Go stack (default recipe).
- **`rust-service`** — the equivalent Rust stack.

## Creating the remote

When `create_remote` is true and no `--remote-url` is given, keel creates the
repository on the provider named by the `provider` answer
(`github`, `gitlab`, `bitbucket`, `sourcecraft`, or `none`) via its REST API.
Credentials always come from the environment — they are never written to disk:

| Provider | Token env (in priority order) | Owner override | Base-URL override |
|----------|-------------------------------|----------------|-------------------|
| `github` | `KEEL_GITHUB_TOKEN`, `GITHUB_TOKEN` | `KEEL_GITHUB_OWNER` | `KEEL_GITHUB_URL` |
| `gitlab` | `KEEL_GITLAB_TOKEN`, `GITLAB_TOKEN` | `KEEL_GITLAB_OWNER` | `KEEL_GITLAB_URL` |
| `bitbucket` | `KEEL_BITBUCKET_TOKEN`, `BITBUCKET_TOKEN` | `KEEL_BITBUCKET_OWNER` | `KEEL_BITBUCKET_URL` |
| `sourcecraft` | `KEEL_SOURCECRAFT_TOKEN`, `SOURCECRAFT_TOKEN` | `KEEL_SOURCECRAFT_OWNER` | `KEEL_SOURCECRAFT_URL` |

The owner defaults to the namespace in your `module_path`
(`gitlab.com/group/subgroup/repo` → `group/subgroup`; `github.com/owner/repo` →
`owner`), overridable per provider with the override variable above. Set
`provider: none` to scaffold a purely local repository.


## Configuration

`keel config` manages a small user-level config at
`$UserConfigDir/keel/config.yaml` so you don't retype your defaults:

```bash
keel config set author.name "Your Name"
keel config set author.email "you@example.com"
keel config set provider github
keel config list
```

Tokens are intentionally **not** stored here — they always come from the
environment.

## Development

The repo uses [Taskfile](Taskfile.yml) for common workflows:

```bash
task           # list available tasks
task build     # build keel into ./bin
task lint      # golangci-lint
task test      # race + shuffled tests
task cover     # coverage profile
task ci        # full local gate (deps + vet + lint + test)
```

Rendered output is verified against golden fixtures under
`internal/render/testdata/golden/`, so changes to any module's templates surface
as a golden diff.

## License

[MIT](LICENSE) © Roman Agaltsev
