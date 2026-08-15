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
- **`--recipe <path>`** — re-apply this recipe instead of the one recorded in
  `.scaffold.lock`.

## Modules the recipe gained

`keel update` re-resolves the recipe, so a module added to it since your repo was
scaffolded is applied like any other change, reported under its own heading:

```text
added module  license              (1 file)
added module  governance           (3 files)
```

A file the new module renders is written when it does not exist. If you already
wrote your own — a hand-rolled `CONTRIBUTING.md`, say — it is treated as any other
edited file: yours is left alone and keel's render is written beside it as
`.keel-new`.

`--modules <csv>` scopes this like everything else, and `--dry-run` previews it.

## Files the recipe no longer produces

When a module stops rendering a file — as `security-go` did when four workflows
became one `security.yml` — keel retracts it **only if it still matches the bytes
keel wrote**:

```text
removed   .github/workflows/codeql.yml       (deleted)
removed   .github/workflows/govulncheck.yml  (edited — left in place)
```

If you edited it, keel never deletes it; it prints an `rm` you can run yourself.
This is the same rule that governs updates: an untouched file is keel's to
change, an edited one is yours.

## Pointing at a moved recipe

A repo scaffolded from a recipe file records where that file was. If it has since
moved, `keel update` says so, updates the modules it already knows about, and
tells you how to do better:

```text
warning: recipe "recipes/my-recipe.yaml" not found; updating only the modules
         recorded in .scaffold.lock.
         Modules added to the recipe since this repo was scaffolded cannot be
         detected. Re-run with --recipe <path> to point at it.
```

`keel update --recipe ./path/to/recipe.yaml` overrides the recorded location.

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

## Upgrading a repo scaffolded before 2.0.0

2.0.0 brings the Go discipline modules up to date with current tooling. Four of
the changes are breaking, and `keel update` cannot make any of them silently — it
never deletes a file it once wrote.

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

  Since 2.1.0 keel **deletes** those for you, as long as they still match the bytes
  keel wrote:

  ```text
  removed   .github/workflows/actionlint.yml         (deleted)
  removed   .github/workflows/codeql.yml             (deleted)
  removed   .github/workflows/dependency-review.yml  (deleted)
  removed   .github/workflows/govulncheck.yml        (deleted)
  removed   main.go                                  (deleted)
  ```

  A file you edited is never deleted — it is listed as `(edited — left in place)`
  and repeated afterwards with an `rm` you can run yourself. Until you do, that one
  keeps firing alongside `security.yml`, on its old action pins.

Branch protection is keel's job since 2.3.0: `.github/keel-settings.yml` declares
the required checks and `keel settings apply` converges them. If you are upgrading
a repo that predates that file, `keel update` adds it — run `keel settings apply`
once afterwards. For a repo whose protection you still manage by hand, the job
names are `lint`, `test`, `typos`, `pr-title`, `actionlint`, `dependency-review`,
plus `codeql` and `govulncheck` when enabled.
