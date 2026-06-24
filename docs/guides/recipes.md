# Recipes

A **recipe** is a named composition of modules — a list of small,
single-purpose modules that `keel` assembles into one repository. You pick a
recipe; `keel` resolves it into its ordered modules, asks their combined
questions, and renders the result.

## The two built-in recipes

`keel` ships two recipes embedded in the binary:

| Recipe | Language | Modules |
|--------|----------|---------|
| `go-service` | go | `base-layout`, `go-mod`, `taskfile-go`, `lint-go`, `test-go`, `security-go`, `dep-bots-go`, `release-go`, `spell` |
| `rust-service` | rust | `base-layout`, `cargo-mod`, `taskfile-rust`, `lint-rust`, `test-rust`, `security-rust`, `release-rust`, `dep-bots-rust`, `spell` |

`go-service` is the default — `keel new` with no `--recipe` uses it. Both share
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
