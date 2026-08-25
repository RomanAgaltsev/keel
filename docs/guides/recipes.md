# Recipes

A **recipe** is a named composition of modules — a list of small,
single-purpose modules that `keel` assembles into one repository. You pick a
recipe; `keel` resolves it into its ordered modules, asks their combined
questions, and renders the result.

## The built-in recipes

`keel` ships these recipes embedded in the binary:

| Recipe | Language | Modules |
|--------|----------|---------|
| `go-service` | go | `base-layout`, `go-mod`, `taskfile-go`, `lint-go`, `test-go`, `security-go`, `dep-bots-go`, `release-go`, `spell`, `license`, `governance`, `contributing-go`, `community-templates` |
| `go-library` | go | same module list as `go-service`, with `archetype: library` — no `cmd/`, no GoReleaser, no `task build` |
| `go-cli` | go | `go-service`'s list plus `cli-go`, with `archetype: cli` — a cobra root command, a `version` subcommand wired to the release ldflags, and a command test |
| `go-analyzer` | go | `go-service`'s list plus `analyzer-go`, with `archetype: cli` — a `go/analysis` analyzer, a `singlechecker` binary, and a working `analysistest` corpus |

The `go-analyzer` corpus under `analyzer/<pkg>/testdata/src/` is GOPATH-shaped
on purpose: that is the layout `analysistest` expects. It is excluded from
`task format`, because `analysistest` matches diagnostics by line and its
fixtures are frequently non-canonical deliberately — being badly formed is
often exactly what the analyzer flags.
| `rust-service` | rust | `base-layout`, `cargo-mod`, `taskfile-rust`, `lint-rust`, `test-rust`, `security-rust`, `release-rust`, `dep-bots-rust`, `spell`, `license`, `governance`, `contributing-rust`, `community-templates` |

`go-service` is the default — `keel new` with no `--recipe` uses it. They share
the language-agnostic `base-layout` and `spell` modules; everything else is the
per-language variant.

## Listing what's available

```bash
keel list
```

`keel list` prints the live set of recipes and modules known to your binary —
the authoritative answer for the version you have installed.

## Running a recipe

```bash
keel new --recipe go-service
keel new --recipe go-library
keel new --recipe rust-service
```

See the per-module breakdown in the
[Module catalog](../reference/module-catalog.md), and
[What you get](../showcase.md) for the directory tree a recipe produces.

## Custom recipes

`--recipe` also accepts a **file path**, so you can compose your own recipe
mixing built-in modules with external ones:

```bash
keel new --recipe ./my-recipe.yaml
```

See **[Authoring → Recipes](../authoring/recipes.md)** for how to write one.
