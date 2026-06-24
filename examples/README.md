# keel examples

Runnable examples for `keel`. Each is exercised by `cmd/keel/examples_test.go`
(via `keel new --no-input --dry-run`), so they stay valid as keel evolves.

## Answers files (`answers/`)

| File | Recipe | What it shows |
|------|--------|---------------|
| `go-service.yaml` | `go-service` | A full Go service, remote created on GitHub. |
| `rust-service.yaml` | `rust-service` | A full Rust crate, remote created on GitHub. |
| `local-only.yaml` | `go-service` | Scaffold locally with no remote (`provider: none`). |
| `ci.yaml` | `go-service` | Minimal set for `--no-input` CI runs (defaults fill the rest). |
| `gitlab.yaml` | `go-service` | Create the remote on GitLab; owner derived from `module_path`. |

Run any of them:

```bash
keel new --recipe go-service --answers examples/answers/local-only.yaml --no-input
```

Add --dry-run to preview the plan without writing anything or hitting the network.

## Custom recipe + external module

| Path | What it shows |
|------|---------------|
| `external-module/` | A worked external module (`module.yaml` + `templates/`): a `bool`+`int` question, a `when` condition, and a templated `.tmpl` file. |
| `custom-recipe/recipe.yaml` | A user recipe composing builtins differently and pulling in the external module via a relative `dir:` source. |
