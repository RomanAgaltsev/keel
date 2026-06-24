# Recipes

A recipe is a YAML file with three fields — `name`, `language`, and `modules`.
The built-in `go-service` and `rust-service` recipes are embedded in the binary,
but `--recipe` also accepts a **file path**, so you can ship your own:

```bash
keel new --recipe ./my-recipe.yaml
```

## Writing one

`modules` is an ordered list. Each entry is either a **bare string** (a built-in
module by name) or a **source-qualified object** (an [external
module](external-modules.md)):

```yaml
name: go-service-lite
language: go
modules:
  - base-layout
  - go-mod
  - taskfile-go
  - lint-go
  - test-go
  - name: editorconfig
    source: { dir: ../external-module }   # relative to this recipe file
```

| Field | Meaning |
|-------|---------|
| `name` | The recipe's name. |
| `language` | `go`, `rust`, or `any` — the recipe's toolchain. |
| `modules` | Ordered module list: built-in names and/or external entries. |

## Language consistency

Every module in a recipe must be language-consistent: a module's `language` has
to be `any` or match the recipe's `language`. A `go` module in a `rust` recipe
(or vice-versa) is rejected — this is what keeps a scaffold's toolchain coherent.

## A worked example

keel ships a complete custom recipe at
[`examples/custom-recipe/recipe.yaml`](https://github.com/RomanAgaltsev/keel/blob/main/examples/custom-recipe/recipe.yaml)
— a leaner Go composition than the built-in `go-service`, mixing built-in module
names with a local external module. Run it against one of the example answers
files:

```bash
keel new --recipe examples/custom-recipe/recipe.yaml \
  --answers examples/answers/local-only.yaml --no-input
```

See [Modules](modules.md) for authoring the modules a recipe composes, and the
normative [Manifest schema](../reference/manifest-schema.md) for the exact field
definitions.
