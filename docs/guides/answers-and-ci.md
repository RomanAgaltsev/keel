# Answers files & CI

Instead of answering the interactive wizard, you can supply every answer in a
YAML file and run `keel` non-interactively — ideal for CI.

## The answers file format

An answers file is flat YAML keyed by question `id`. The **core questions**
(asked for every recipe):

| Key | Type | Meaning |
|-----|------|---------|
| `repo_name` | string | Repository / directory name. |
| `description` | string | One-line project description. |
| `module_path` | string | Go module path / canonical import path (`github.com/you/demo`). Also drives provider owner derivation. |
| `author_name` | string | Commit author name. |
| `author_email` | string | Commit author email. |
| `license` | string | License identifier (e.g. `MIT`). |
| `visibility` | string | `public` or `private`. |
| `provider` | string | `github`, `gitlab`, `bitbucket`, `sourcecraft`, or `none`. |
| `create_remote` | bool | Whether to create the remote repository. |

```yaml
# answers.yaml
repo_name: demo
description: a demo service
module_path: github.com/you/demo
author_name: Your Name
author_email: you@example.com
license: MIT
visibility: public
provider: github
create_remote: true
```

Each module also contributes its own questions; those merge in **by `id`**. Any
module question you omit falls back to its default under `--no-input`. Ready-made
answers files for several scenarios live in
[`examples/answers/`](https://github.com/RomanAgaltsev/keel/tree/main/examples/answers)
(`ci.yaml`, `gitlab.yaml`, `local-only.yaml`, `go-service.yaml`,
`rust-service.yaml`).

## The relevant flags

- **`--answers <file>`** — read answers from the YAML file.
- **`--no-input`** — never prompt. A missing **required** answer is an error
  (this is what makes a run safe for CI); optional module questions use their
  defaults.
- **`--dry-run`** — print the plan only. Touches neither disk nor network, so
  it's safe to run anywhere.

## A CI snippet

```bash
keel new --recipe go-service --answers examples/answers/ci.yaml --no-input
```

`examples/answers/ci.yaml` uses `provider: none` and `create_remote: false`, so
it scaffolds and commits locally without touching any remote — exactly what you
want in a pipeline that just verifies scaffolding works.
