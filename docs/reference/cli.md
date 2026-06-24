# CLI

Every `keel` command and its flags. This page is hand-maintained against the
binary — run `keel <command> --help` for the authoritative, version-specific
list.

## `keel new`

Scaffold a repository from a recipe.

```bash
keel new --recipe go-service
```

| Flag | Default | Effect |
|------|---------|--------|
| `--recipe` | `go-service` | Recipe to use — a built-in name or a path to a recipe file. |
| `--target` | repo name | Target directory. |
| `--answers` | — | Read answers from a YAML file. |
| `--no-input` | `false` | Never prompt; fail if a required answer is missing (CI mode). |
| `--remote-url` | — | Wire/clone an existing remote instead of creating one. |
| `--overwrite` | `false` | Overwrite existing files when overlaying onto a tree. |
| `--dry-run` | `false` | Print the plan; touch neither disk nor network. |

## `keel update`

Re-apply evolved module templates to an existing repo (hash-aware overlay; user
edits are preserved as `.keel-new`). See [Updating a repo](../guides/updating.md).

| Flag | Default | Effect |
|------|---------|--------|
| `--path` | `.` | Repository path to update. |
| `--dry-run` | `false` | Print the plan; write nothing (external module sources are still fetched). |
| `--reconfigure` | `false` | Re-run the wizard and re-render all modules. |
| `--no-input` | `false` | Never prompt (CI mode); only meaningful with `--reconfigure`. |
| `--commit` | `false` | Commit the update when there are no conflicts (`chore: keel update`). |
| `--overwrite` | `false` | Overwrite user-edited files instead of writing `.keel-new`. |
| `--modules` | — | Restrict to a comma-separated subset of modules. |

## `keel outdated`

Report which of a repo's keel modules have newer versions available, without
changing anything.

| Flag | Default | Effect |
|------|---------|--------|
| `--path` | `.` | Repository path to inspect. |
| `--modules-only` | `false` | Check only keel module versions. |
| `--tools-only` | `false` | Check only tool/action pins. |

## `keel list`

List the available recipes and modules known to your binary.

```bash
keel list
```

## `keel config`

Manage keel's user config (`$UserConfigDir/keel/config.yaml`). See
[Configuration](../guides/config.md).

| Subcommand | Effect |
|------------|--------|
| `keel config get <key>` | Print a single config value. |
| `keel config set <key> <value>` | Set a config value. |
| `keel config list` | Print all config values. |

A persistent `--file <path>` flag points the subcommands at a config file in a
specific location.

## `keel version`

Print version, commit, and build date:

```bash
keel version
# keel 1.7.1 (commit abc1234, built 2026-06-23)
```
