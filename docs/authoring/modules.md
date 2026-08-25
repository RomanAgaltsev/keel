# Modules

A **module** is the unit of composition in keel: a directory with a
`module.yaml` manifest and a `templates/` tree. A recipe is just an ordered list
of modules, so authoring a module is how you extend keel with a new capability.

```text
my-module/
├── module.yaml
└── templates/
    ├── .github/workflows/my-thing.yml.tmpl
    └── config.toml
```

## The manifest

Every field of `module.yaml`, with a worked example:

```yaml
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
    when: "{{ .enable_codeql }}"   # optional condition gating the file
  - src: ".github/workflows/dependency-review.yml"
    dest: "."
```

| Field | Meaning |
|-------|---------|
| `name` | Unique module name. Used in recipes and as the collision key — two modules can't share a name. |
| `description` | One-line summary (shown by `keel list`). |
| `version` | Semantic version. **Bump it whenever the templates change** — `task modules:check` enforces this in the keel repo so an unbumped change fails CI. The version is recorded in `.scaffold.lock` and drives `keel outdated` / `keel update`. |
| `language` | `any`, `go`, or `rust`. Enforced against the recipe's language — a `go` module can't appear in a `rust` recipe. Use `any` for language-agnostic modules. |
| `requires` | Other modules this one depends on, so a recipe stays consistent (e.g. most modules `requires: [base-layout]`). |
| `questions` | The module's own questions (see below). |
| `files` | The render rules (see below). |
| `deps` | Third-party Go modules your emitted code imports (see below). |

## Dependencies

If your module emits Go code that imports something outside the standard
library, declare it. keel unions every module's `deps` across the recipe and the
`go-mod` module renders the `require` block, so you never edit `go.mod`
yourself — and two modules needing the same dependency don't collide.

```yaml
deps:
  - path: golang.org/x/tools
    version: v0.38.0
```

Three rules:

- **Declare direct imports only.** The scaffolded repo runs `go mod tidy` as the
  first step of its own `task ci`, which resolves the indirect closure and
  writes `go.sum`.
- **Versions are `v`-prefixed semver.** `v0.38.0`, not `0.38.0`. Put the version
  in `version`, never as `path@version`.
- **Go modules only.** Declaring `deps` on an `any` or `rust` module is an
  error. Cargo support is unbuilt until a Rust module needs it.

If two modules in a recipe declare the same path at different versions, the
higher one wins — the same rule Go's own minimal version selection applies.

## Questions

Each entry adds one question, merged with the core questions and other modules'
questions by `id`:

| Key | Meaning |
|-----|---------|
| `id` | Answer key. Templates reference it as `{{ .id }}`; answers files key on it. |
| `prompt` | Text shown in the interactive wizard. |
| `type` | One of `string`, `bool`, `select`, `multiselect`, `int`. |
| `options` | The allowed choices, for `select` / `multiselect`. |
| `default` | Value used when unanswered (and under `--no-input` for optional questions). |

## Files

Each `files` entry maps templates into the rendered repo:

| Key | Meaning |
|-----|---------|
| `src` | A glob relative to the module's `templates/` directory. |
| `dest` | Destination directory in the rendered repo (`.` is the repo root). |
| `when` | Optional `text/template` condition; the file is rendered only when it evaluates truthy. |

A file whose name ends in **`.tmpl`** is rendered as a Go `text/template` (the
`.tmpl` suffix is stripped from the output name). Any other file is **copied
verbatim**. Templates see the full merged answer set — `{{ .repo_name }}`,
`{{ .module_path }}`, your module's own question ids, and so on.

Rendering uses `missingkey=error`: referencing a key that nobody answered fails
the run loudly rather than emitting an empty string, so a typo in a template
surfaces immediately.

## Going further

- A normative, field-by-field reference lives in
  [Reference → Manifest schema](../reference/manifest-schema.md).
- To ship a module *outside* the keel binary, see
  [External modules](external-modules.md).
- To compose modules into your own recipe, see [Recipes](recipes.md).

## If your module ships workflows

A module whose `templates/` tree contains `.github/workflows/` needs its
templates directory added to the `github-actions` block in keel's own
`.github/dependabot.yml`, or its action pins will never be updated. See
[CONTRIBUTING.md](https://github.com/RomanAgaltsev/keel/blob/main/CONTRIBUTING.md).
