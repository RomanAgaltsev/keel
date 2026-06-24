# Repo states

`keel` is state-aware: before writing anything it detects whether the **target
directory** already exists locally and whether the **remote** already exists,
then branches accordingly. It is never just a file dump.

## The four-state matrix

| Local | Remote | Behavior |
|-------|--------|----------|
| absent | absent | **Fresh scaffold** — atomic write, `git init`, first commit, and (if requested) create + push the remote. |
| present | absent | **Overlay** onto the existing tree (skip-existing), then optionally create + push the remote. |
| absent | present | **Clone-then-overlay** — clone the remote, overlay the rendered files on top, commit. |
| present | present | **Overlay, then stop.** keel overlays locally but, with both sides already carrying history, refuses to reconcile for you — it prints the `git fetch` / merge / `push` steps to run yourself. |

**keel never force-pushes.** When both local and remote have history, it hands
reconciliation back to you rather than overwriting either side.

## Skip-existing vs `--overwrite`

When overlaying onto a directory that already has files, the default is
**skip-existing**: keel writes the files that aren't there yet and leaves any
existing file untouched. Pass **`--overwrite`** to replace existing files with
the freshly rendered versions instead.

Either way, **keel never deletes a file it didn't write.** Removing capabilities
is your call, not keel's.

To re-apply *evolved* module templates to a repo keel already scaffolded — as
opposed to overlaying a fresh recipe — use [`keel update`](updating.md), which
is hash-aware and preserves your edits.
