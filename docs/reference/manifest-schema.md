# Manifest schema

The normative field-by-field reference for `module.yaml` and recipe files. For a
gentler, example-driven walkthrough see
[Authoring → Modules](../authoring/modules.md) and
[Authoring → Recipes](../authoring/recipes.md).

## `module.yaml`

| Field | Type | Required | Meaning |
|-------|------|----------|---------|
| `name` | string | yes | Unique module name. |
| `description` | string | yes | One-line summary. |
| `version` | string (semver) | yes | Bumped when templates change; recorded in the lockfile. |
| `language` | enum | yes | `any`, `go`, or `rust`. Must be `any` or match the recipe's language. |
| `requires` | string[] | no | Names of modules this one depends on. |
| `questions` | question[] | no | The module's questions. |
| `files` | file[] | no | The render rules. |

### `questions[]`

| Field | Type | Required | Meaning |
|-------|------|----------|---------|
| `id` | string | yes | Answer key; referenced in templates as `{{ .id }}`. |
| `prompt` | string | yes | Wizard prompt text. |
| `type` | enum | yes | `string`, `bool`, `select`, `multiselect`, or `int`. |
| `options` | string[] | for `select` / `multiselect` | Allowed choices. |
| `default` | any | no | Value used when unanswered. |

### `files[]`

| Field | Type | Required | Meaning |
|-------|------|----------|---------|
| `src` | string (glob) | yes | Glob relative to the module's `templates/` dir. |
| `dest` | string | yes | Destination directory in the rendered repo (`.` = root). |
| `when` | string (`text/template`) | no | Condition; the file renders only when it evaluates truthy. |

Files ending in `.tmpl` are rendered as Go `text/template` (suffix stripped);
all others are copied verbatim. Rendering uses `missingkey=error`.

## Recipe file

| Field | Type | Required | Meaning |
|-------|------|----------|---------|
| `name` | string | yes | Recipe name. |
| `language` | enum | yes | `any`, `go`, or `rust`. |
| `modules` | module-ref[] | yes | Ordered list of modules. |

### `modules[]` (module-ref)

Each entry is one of:

- a **bare string** — a built-in module by name; or
- an **object** — `{ name: <string>, source: <source> }` for an external module.

A `source` has exactly one of:

| Field | Meaning |
|-------|---------|
| `dir` | Filesystem path to the module, relative to the recipe file. |
| `git` | Repository URL. Combined with `subdir` (path within the repo) and `ref` (tag, branch, or commit). |

See [External modules](../authoring/external-modules.md) for source semantics and
caching.
