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
