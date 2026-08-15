# Recipes

A recipe is a YAML file with four fields — `name`, `language`, `archetype`, and
`modules`. The built-in recipes are embedded in the binary,
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
| `archetype` | `service` (default), `library`, or `cli` — whether the repo produces a binary. |
| `modules` | Ordered module list: built-in names and/or external entries. |

## Archetypes

`archetype` tells the modules whether the repo produces a binary. Omit it and you
get `service`, which is what every recipe before this field did.

A `library` recipe drops the `cmd/<name>` entrypoint, the Taskfile's `build` task
and its ldflags, the `task build` lines in README and CONTRIBUTING,
`.goreleaser.yaml`, and the `goreleaser` job in `release.yml`. It keeps
release-please, so a library is still tagged and still gets a changelog. In its
place `go-mod` emits `doc.go`, `<name>.go` and `<name>_test.go`.

`cli` behaves exactly like `service` today — a CLI is a binary. It exists so that
recipes can declare what they are.

Templates never test `archetype` directly. Rendering derives two more keys from
it, and those are what modules use:

| Key | Meaning |
|-----|---------|
| `is_library` | `true` only for `archetype: library`; every `when:` and template guard uses this |
| `package_name` | `repo_name` reduced to a legal Go package name (`go-thing` → `gothing`) |

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
