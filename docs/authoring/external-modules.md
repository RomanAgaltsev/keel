# External modules

You don't have to fork keel to ship a module. **External modules** are
referenced by source-qualified entries in a **user-supplied recipe file** — there
is no central registry. keel resolves them at run time, alongside its built-in
modules.

## The two source kinds

A recipe module entry is either a bare string (a built-in module) or an object
with a `name` and a `source`. A source is exactly one of `dir` or `git`:

```yaml
modules:
  - base-layout              # builtin
  - name: editorconfig       # external, from a local directory
    source: { dir: ../external-module }   # path relative to the recipe file
  - name: logging            # external, from git
    source:
      git: https://github.com/you/keel-mods.git
      subdir: logging        # path to the module within the repo
      ref: v1.2.0            # tag, branch, or commit
```

- **`dir:`** — a filesystem path **relative to the recipe file**.
- **`git:`** — a repository URL, with `subdir` locating the module inside it and
  `ref` pinning the version.

In both cases the module's filesystem is **rooted at the module directory**:
`module.yaml` and `templates/` sit at the root of the `dir` / `subdir`, not under
`modules/<name>/` the way the built-in modules are laid out.

## Git caching and pinning

Git sources are cloned into the user cache directory and **pinned to the resolved
commit SHA**, which is recorded in `.scaffold.lock` so a later `keel update`
re-renders from exactly the same source.

!!! warning "Prefer tags or SHAs over branches"
    A branch `ref` is cached and **not refreshed** on later runs — keel reuses
    the cached commit. To track a moving target you'd have to clear the cache.
    Pin to a tag or commit SHA for reproducible, intentional updates.

## Rules and safety

- **No shadowing.** A name that collides with a built-in module — or with
  another external module in the same recipe — is an error. Names are unique.
- **`requires` resolves narrowly.** An external module's `requires` are
  satisfied only from the built-in modules plus the modules listed in the same
  recipe.
- **Path-traversal guard.** Any rendered `dest` that would escape the target
  directory is rejected.

## A worked example

keel ships a complete external module you can copy:
[`examples/external-module/`](https://github.com/RomanAgaltsev/keel/tree/main/examples/external-module)
— a templated `.editorconfig` with `module.yaml` and `templates/` at the
directory root, wired up by
[`examples/custom-recipe/recipe.yaml`](https://github.com/RomanAgaltsev/keel/blob/main/examples/custom-recipe/recipe.yaml)
via a `dir:` source. See [Recipes](recipes.md) for the recipe side.
