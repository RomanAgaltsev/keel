# keel

Scaffold a new git repository from composable template modules.

`keel` turns a *recipe* — a named list of small, single-purpose modules — into a
ready-to-push repository: source layout, Taskfile, CI workflows, linter and
release config, all rendered from your answers. It then `git init`s, writes the
first commit, and (optionally) creates the remote and pushes — in one command.

```bash
keel new --recipe go-service
```
## Where to go next

- **[Install](getting-started/install.md)** and **[Quick start](getting-started/quickstart.md)**.
- **[Your first repo](getting-started/first-repo.md)** — a guided walkthrough.
- **[Recipes](guides/recipes.md)**, **[Providers](guides/providers.md)**, and
  **[Updating a repo](guides/updating.md)**.
- **[Authoring modules](authoring/modules.md)** to extend keel.
