# Example custom recipe: `go-service-lite`

A user-supplied recipe file (pass its path to `--recipe`). It shows two things:

1. **Custom composition** — a leaner Go stack than the builtin `go-service`.
2. **An external module** — `editorconfig` is pulled from `../external-module`
   via a `dir:` source (resolved relative to this file).

```bash
keel new --recipe examples/custom-recipe/recipe.yaml \
  --answers examples/answers/local-only.yaml --no-input
```
