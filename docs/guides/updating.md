# Updating a repo

Modules evolve. `keel update` re-renders a repo's modules at their current
template versions and overlays the result onto your repository, using the
per-file hashes recorded in `.scaffold.lock` to tell *your* edits apart from
keel's.

```bash
keel update
```

## How a file is classified

For each file a module would render, `keel update` compares the file on disk
against the hash recorded at scaffold time:

- **Untouched file** (on-disk hash matches the lock) → **updated in place** with
  the new render.
- **User-edited file** (on-disk content differs from the recorded hash) →
  **preserved**. The new render is written alongside as `<path>.keel-new` for
  you to merge by hand.
- **Removed file** (a file the module no longer renders) → **reported**, never
  deleted. Removing it is your decision.

Pass **`--overwrite`** to replace user-edited files in place instead of writing
`.keel-new` sidecars.

## Flags

- **`--dry-run`** — preview the classification (which files would be updated,
  preserved, or reported) without writing anything. External module sources are
  still fetched so the preview is accurate.
- **`--reconfigure`** — re-run the wizard and re-render all modules with fresh
  answers.
- **`--commit`** — when there are no conflicts, make a single
  `chore: keel update` commit of keel's own changed files.
- **`--modules <csv>`** — restrict the update to a comma-separated subset of
  modules.
- **`--path <dir>`** — the repository to update (defaults to the current
  directory).

## Older scaffolds

A repo scaffolded by an older keel still updates cleanly: newly-added question
**defaults** are filled in automatically. But if a new **required** question has
no default, keel can't guess — it asks you to re-run with `--reconfigure` so you
can answer it.

A v1 lockfile (no per-file hashes) is read transparently and upgraded to v2 on
the first update — see [Lockfile](../reference/lockfile.md).

## Checking without changing anything

```bash
keel outdated
```

`keel outdated` reports which of a repo's modules have a newer version available
— and nothing else. It writes no files. Use it to decide whether an update is
worth running.

## Upgrading a repo scaffolded before 1.8.0

1.8.0 actualizes the Go discipline modules against what the vault's shipped repos
actually run. Four of the changes are breaking, and `keel update` cannot make any
of them silently — it never deletes a file it once wrote.

- **The entrypoint moved to `cmd/<name>/main.go`.** `go-mod` emits the new path;
  the old root `main.go` is reported as no longer produced. Move your code across,
  then delete it. `Taskfile.yml` and `.goreleaser.yaml` both target `./cmd/<name>`
  now, so a release built before you move will produce nothing.
- **`goimports` is replaced by `gci`.** `.golangci.yml` now enables `gofumpt` +
  `gci` as formatters, with `gci` sections seeded to `standard`, `default` and
  `prefix(<your module path>)`. Run `task format` once and commit the import
  reordering as its own commit.
- **`depguard` now gates dependencies.** The allow-list starts at `$gostd` plus
  your own module path, so the first `go get` fails lint until you add the new
  dependency to `linters.settings.depguard.rules.Main.allow`. That is deliberate —
  it makes a new dependency a decision rather than an accident. Existing
  third-party imports must be added in the upgrade commit.
- **Four security workflows became one.** `codeql.yml`, `govulncheck.yml`,
  `dependency-review.yml` and `actionlint.yml` are replaced by `security.yml`.
  `keel update` prints the exact commands:

  ```text
  5 file(s) are no longer produced by keel and were left in place:

      rm .github/workflows/actionlint.yml
      rm .github/workflows/codeql.yml
      rm .github/workflows/dependency-review.yml
      rm .github/workflows/govulncheck.yml
      rm main.go
  ```

  Until you run them the old workflows keep firing alongside the new one, on their
  old action pins.

If branch protection required the old checks by name, update it: the job names are
now `lint`, `test`, `typos`, `pr-title`, `actionlint`, `dependency-review`, plus
`codeql` and `govulncheck` when enabled.
