# Repository settings

Scaffolding a repo gets the files right. It does not get the *remote* right — branch
protection, security toggles, the Actions policy and the merge policy all live on the
hosting provider, and used to be a checklist in `CONTRIBUTING.md` that every maintainer
worked through by hand, differently.

keel renders those into `.github/keel-settings.yml` and applies them.

## The file

```yaml
version: 1

repository:
  allow_squash_merge: true
  allow_merge_commit: false
  delete_branch_on_merge: true

security:
  dependabot_alerts: true
  private_vulnerability_reporting: true

actions:
  allowed: local_and_verified
  allowed_patterns:
    - arduino/setup-task@*
    - crate-ci/typos@*
  default_workflow_permissions: read
  can_approve_pull_request_reviews: true

rulesets:
  - name: "keel: main"
    target: branch
    ref: main
    required_status_checks: [lint, test, typos, pr-title, dependency-review, actionlint]
    required_linear_history: true
    block_force_push: true
    bypass: [repo_admin]
```

**Only the keys the file names are converged.** An omitted key is left exactly as it is,
so this is safe to run against a repository you have also tuned by hand. That is why
every field is optional: `has_wiki: false` and an absent `has_wiki` mean different things —
"turn it off" versus "not keel's business".

`.github/keel-settings.yml` is deliberately *not* `.github/settings.yml`, which the Probot
Settings app watches. The two would fight over the same file.

### Two lists you should not hand-edit

`required_status_checks` and `actions.allowed_patterns` are **computed from the
modules the recipe includes**. Each module declares the checks it reports and
the actions it uses in its own `emits` block (see
[the manifest schema](../reference/manifest-schema.md#emits)), and keel renders
the union.

Editing either by hand in a scaffolded repo works until the next
`keel new`/`keel update` regenerates the file. Change the module set, or the
module's own `emits` block, instead.

The two lists are why this matters more than tidiness:

- a required check nothing reports leaves every pull request `BLOCKED` with all
  runs green, and
- an action outside `allowed_patterns` makes its workflow fail to *start* —
  `startup_failure` at 0s, no check reported, and it cannot be re-run, so
  recovering needs a fresh push.

`allowed_patterns` extends `local_and_verified` rather than replacing it: the
policy stays "GitHub-owned, verified, **and** these named third parties". It is
only valid alongside `allowed: local_and_verified`; `all` needs no list and
`local_only` admits none.

`can_approve_pull_request_reviews` is on when a module in the recipe asks for it
— release-please and release-plz both need it to open their release PRs.
`default_workflow_permissions` stays `read` regardless.

## Applying it

```bash
keel settings apply           # converge the remote to the file
keel settings apply --check   # report drift, change nothing
```

`keel new` also applies it once, at the very end, after the remote exists and the code is
pushed. That step can never fail a scaffold: if the token lacks scope or a group errors,
you get a repo, a report, and a `keel settings apply` command to retry with.

`task keel:settings` in a scaffolded repo runs the `--check` form.

### Exit codes

| Command | 0 | 1 | 2 |
|---|---|---|---|
| `apply` | everything applied | a group failed | the file or target could not be read |
| `apply --check` | in sync | drift found | a group could not be read |

`--check` separates "differs" from "could not look". A check that cannot reach the remote
must not be mistaken for a clean one, which is why the latter is exit 2.

### Targeting

The repository and provider come from `.scaffold.lock`. For a repo keel never created,
name it:

```bash
keel settings apply -C ./some-repo --repo owner/name --provider github
```

## Token scope

`administration:write` on the repository (a classic token's `repo` scope covers it).
Without it, groups fail individually and are reported; nothing else is affected.

## Sharp edges

**A required status check that never reports blocks every pull request.** The
`required_status_checks` list must match the *job* names in `.github/workflows/`. This
fails quietly: repository admins are waved through by the `repo_admin` bypass, so the
person most likely to notice is the one least likely to be blocked.

The rendered list adapts to your *answers* — turning off `enable_codeql` drops `codeql` —
but it cannot see which **modules** your recipe selected. A custom recipe that takes
`repo-settings-go` without `security-go` still lists `dependency-review` and `actionlint`,
which nothing will then report. **Remove those two lines by hand**, or keep `security-go`.
The stock `go-service` and `rust-service` recipes are always consistent.

**Rulesets need GitHub Pro on a private repository.** On a free plan keel reports the
ruleset as *unsupported* — not failed — converges everything else, and moves on.

**An unreachable repository shows as drift on the toggle settings.** GitHub answers `404`
both for "this toggle is off" and for "this repository is not visible to you", and the two
cannot be told apart from the response. keel reads it as *off*, so pointing `--repo` at a
name that does not exist reports `security.private_vulnerability_reporting: false -> true`
alongside real failures from every other group. Read the failures, not the drift.

**Topics are replace-all.** Declaring `topics` hands keel ownership of the whole list;
omit it to keep hand-managed topics.

**Secret scanning is unavailable on private repos** without GitHub Advanced Security. The
rendered file declares it only when the repo is public.

## Other providers

GitHub only, for now. A provider without settings support is reported as such and the
scaffold completes — GitLab, Bitbucket and SourceCraft mappings are designed but
deliberately unbuilt until there is a real repository on each host to verify them against.
