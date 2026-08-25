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
| `emits` | emits | no | What this module contributes to the repository's CI contract. |
| `deps` | dep[] | no | Third-party Go modules this module's emitted code imports. |

### `deps[]`

The direct third-party imports of the code this module emits. keel takes the
union across the recipe and `go-mod` renders the `require` block, so the module
that needs a dependency stays decoupled from the module that owns `go.mod`.

| Field | Type | Required | Meaning |
|-------|------|----------|---------|
| `path` | string | yes | Module path, e.g. `golang.org/x/tools`. No `@version` — that goes in `version`. |
| `version` | string (semver) | yes | `v`-prefixed, e.g. `v0.38.0`. |

Rules:

- **Direct imports only.** `go mod tidy` — which a scaffolded repo runs as the
  first step of its own `task ci` — computes the indirect closure and writes
  `go.sum`.
- **Go modules only.** A `deps` block on a module whose `language` is not `go`
  is an error, not a silent no-op. Cargo support is unbuilt because no Rust
  consumer exists yet.
- **Two modules may declare the same path.** The higher version wins, matching
  Go's minimal version selection.

### `emits`

The facts a sibling module — in practice `repo-settings-*` — would otherwise
have to know about this one by hand, and get wrong when this one changes. keel
takes the union across a recipe and the settings module renders it, so nothing
restates anything.

| Field | Type | Required | Meaning |
|-------|------|----------|---------|
| `checks` | string[] | no | Status-check contexts this module's workflows report, named exactly as the host names them. |
| `actions` | string[] | no | Third-party actions its workflows use, **without a version**. |
| `needs` | string[] | no | Repository capabilities its workflows require. Closed vocabulary. |

An absent block means the module contributes nothing, which is right for a
module that emits no workflows.

```yaml
emits:
  checks: [lint]
  actions: [arduino/setup-task]
```

**`checks` and matrixed jobs.** A matrixed job reports one context per cell —
`test (ubuntu-latest)`, not `test` — so declaring the job's bare name would
require a context nothing ever reports, and every pull request would block with
all runs green. A module that wants one stable context adds an aggregate gate
job and names *that*. A guard enforces this: every declared check must have an
unmatrixed job of the same name.

**`actions` and versions.** Name `arduino/setup-task`, not
`arduino/setup-task@v3` — the allow-list is rendered as `owner/repo@*`, so a
version here would be duplicated precision that drifts against the workflow's
own pin. Actions under `actions/` and `github/` are GitHub-owned, always
permitted, and need no declaration. A Docker action keeps its `docker://`
prefix and is allow-listed bare, because it is referenced as
`docker://image:tag` rather than with `@`.

**`needs` vocabulary.** Currently only `can_approve_pull_request_reviews`,
required by any module whose workflows open a pull request with the default
`GITHUB_TOKEN` — release-please and release-plz both do. The list is closed: an
unrecognised entry fails the build rather than being silently dropped, which is
the failure this whole mechanism exists to remove.

**External modules.** `emits` is optional, so an external module that omits it
contributes nothing to the allow-list — and a workflow of its own using an
unverified third-party action will be blocked by the policy until it declares
that action.

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
| `user_owned` | bool | no | The file's content becomes the user's once they edit it. |

A `user_owned` file behaves in three ways:

- **Untouched** — it updates normally (`Clean`), so keel keeps actualizing what
  it owns. This is what lets `keel update --reconfigure` still rewrite
  `go.mod`'s module path.
- **Edited** — it is skipped silently. No `Conflict`, and crucially no
  `.keel-new` sidecar, which would carry keel's render *without* the user's
  additions and be dangerous to act on.
- **Retracted** — never. If its module leaves the recipe, the file stays.

It remains *recorded* in the lockfile, so its baseline is still available.

`go.mod` is the motivating case: the first `go get` adds a `require` line keel
did not write, but the `module` line stays keel's to actualize.

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
