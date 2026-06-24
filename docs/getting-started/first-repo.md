# Your first repo

This walks through a single interactive run of:

```bash
keel new --recipe go-service
```

Under the hood, a `keel new` run is a small state machine. Here's what happens,
step by step:

1. **Resolve the recipe.** `keel` looks up `go-service`, expands it into its
   ordered list of modules (`base-layout`, `go-mod`, `taskfile-go`, `lint-go`,
   `test-go`, `security-go`, `dep-bots-go`, `release-go`, `spell`), and loads
   each module's manifest.

2. **Ask the questions.** The built-in core questions (repo name, module path,
   author name and email, license, visibility, provider) are merged with each
   module's own questions, then asked in a typed wizard. Press through the
   prompts — defaults come from your [config](../guides/config.md) and
   `git config` where available.

3. **Build the render plan.** Every module's `files` rules are expanded —
   templated with Go `text/template` and gated by optional `when` conditions —
   into one combined set of destination files. If two modules would write the
   same path, the plan fails early rather than silently clobbering.

4. **Detect state.** `keel` checks whether the target directory already exists
   and whether the remote already exists, then chooses how to materialize (see
   **[Repo states](../guides/repo-states.md)**). For a brand-new repo, both are
   absent — a fresh scaffold.

5. **Materialize.** The plan is written to disk. For a fresh repo this is a
   single atomic write; `keel` never deletes a file it didn't create.

6. **git init + commit.** `keel` initializes git on branch `main`, sets the
   author identity, writes `.scaffold.lock` (recording the recipe, modules,
   their versions, your answers, and per-file hashes), stages everything, and
   makes the first commit `chore: scaffold with keel`.

7. **Create the remote (optional).** If you answered `create_remote: true` and a
   `provider`, `keel` creates the repository on that provider, wires `origin`,
   and pushes. When both local and remote already have history, `keel` refuses
   to force anything and prints the fetch/rebase/push steps for you to run.

8. **Summary.** `keel` prints what it did and any next steps.

## What landed

You now have a ready-to-push Go service: a module layout and entrypoint, a
`Taskfile`, golangci-lint v2, a race/coverage test workflow, CodeQL +
govulncheck + dependency-review security scans, Dependabot/Renovate, a
release-please + GoReleaser pipeline, and a typos spell-check — all committed.

See **[What you get](../showcase.md)** for the full directory tree a
`go-service` scaffold produces.
