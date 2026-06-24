# Quick start

Four ways to drive `keel new`, from interactive to fully hands-off:

```bash
# Interactive: keel asks for repo name, module path, author, license, etc.
keel new --recipe go-service

# See what would happen without writing anything or hitting the network.
keel new --recipe go-service --dry-run

# Non-interactive (CI): take every answer from a file, never prompt.
keel new --recipe go-service --answers answers.yaml --no-input

# Scaffold a Rust crate instead.
keel new --recipe rust-service
```

## The answers file

An answers file mirrors the question IDs `keel` would otherwise ask
interactively. The core keys:

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

Module-specific questions merge in by their `id`; any you omit fall back to
their defaults under `--no-input`. Ready-to-run answers files (CI, GitLab,
local-only, and per-recipe) live in
[`examples/answers/`](https://github.com/RomanAgaltsev/keel/tree/main/examples/answers).

See **[Answers files & CI](../guides/answers-and-ci.md)** for the full question
reference and a copy-paste CI snippet, or
**[Your first repo](first-repo.md)** for a guided walkthrough of an interactive
run.
