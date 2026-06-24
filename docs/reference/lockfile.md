# Lockfile

`keel` writes a `.scaffold.lock` at the root of every repository it scaffolds.
It records exactly what was rendered — recipe, modules and their versions, the
answers, and a hash of every rendered file — so a run is reproducible and
[`keel update`](../guides/updating.md) can tell your edits apart from keel's.

## The v2 shape

```yaml
lock_version: 2
keel_version: 1.7.1
recipe: go-service
modules:
  - name: base-layout
    source: builtin
    version: 1.0.0
    files:
      - path: README.md
        sha256: 0c1f…
      - path: .gitignore
        sha256: 9ab2…
  - name: editorconfig
    source: dir:../external-module
    version: 1.0.0
    files:
      - path: .editorconfig
        sha256: 4d7e…
answers:
  repo_name: demo
  module_path: github.com/you/demo
  # …every answer, including module questions
```

| Field | Meaning |
|-------|---------|
| `lock_version` | Schema version (`2`). |
| `keel_version` | The keel version that wrote the lock. |
| `recipe` | Recipe name (or the path passed to `--recipe`). |
| `modules[]` | One entry per module, in render order. |
| `modules[].name` | Module name. |
| `modules[].source` | Provenance: `builtin`, `dir:<path>`, or `git:<url>//<subdir>@<ref>`. |
| `modules[].version` | The manifest `version` for builtin/dir modules; the resolved commit SHA for git sources. |
| `modules[].files[]` | Per-file render hashes — `{ path, sha256 }` — keyed by destination path. |
| `answers` | Every answer used, so `keel update` can re-render without re-asking. |

The per-file `sha256` values are how `keel update` classifies a file as
untouched (matches the lock → update in place) or user-edited (differs →
preserve, write `.keel-new`).

!!! note "Tokens are never recorded"
    `answers` holds your scaffold answers, not secrets. Provider tokens come
    from the environment and are **never** written to the lock — see
    [Providers](../guides/providers.md).

## v1 locks

A v1 lockfile predates per-file hashes (no `files` under each module). keel
reads v1 locks transparently and upgrades them to v2 on the first `keel update`.
