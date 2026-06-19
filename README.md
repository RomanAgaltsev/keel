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

## Why keel?

- **Composition over monoliths.** A repository is assembled from independent
  modules (`base-layout`, `go-mod`, `lint`, `release`, …). Add a capability by
  adding a module to a recipe, not by editing a giant template.
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
```

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
# modules/security/module.yaml
name: security
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
dependencies so a recipe stays consistent.

### Recipes

A recipe is just a named composition of modules:

```yaml
# recipes/go-service.yaml
name: go-service
language: go
modules: [base-layout, go-mod, taskfile, lint, test, security, dep-bots, release, spell]
```

## Built-in modules

| Module | Description |
|--------|-------------|
| `base-layout` | README and `.gitignore` common to every repo |
| `go-mod` | Minimal Go module and entrypoint |
| `taskfile` | Taskfile with project-local `bin/` tooling and a CI gate |
| `lint` | golangci-lint v2 config + lint workflow |
| `test` | race/shuffle test workflow with coverage |
| `security` | CodeQL, govulncheck, dependency-review, workflow linting |
| `dep-bots` | Dependabot or Renovate dependency-update config |
| `release` | release-please + GoReleaser release pipeline |
| `spell` | Spell-check with crate-ci/typos |

The bundled **`go-service`** recipe composes all of them into a production-ready
Go service.

## Creating the remote

When `create_remote` is true and no `--remote-url` is given, keel creates the
repository on GitHub via the REST API. Credentials come from the environment —
they are never written to disk:

| Variable | Purpose |
|----------|---------|
| `KEEL_GITHUB_TOKEN` (or `GITHUB_TOKEN`) | API token used to create/inspect the repo |
| `KEEL_GITHUB_OWNER` | Repo owner; otherwise derived from the module path (`github.com/<owner>/<repo>`) |

Set `provider: none` (or answer accordingly) to scaffold a purely local
repository.

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
