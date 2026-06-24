# Providers

When `create_remote` is true and you don't pass `--remote-url`, `keel` creates
the repository on the provider named by your `provider` answer, via its REST
API, then wires `origin` and pushes.

## The four providers

`provider` accepts `github`, `gitlab`, `bitbucket`, `sourcecraft`, or `none`.
Credentials and overrides always come from the **environment** — never from a
file:

| Provider | Token env (in priority order) | Owner override | Base-URL override |
|----------|-------------------------------|----------------|-------------------|
| `github` | `KEEL_GITHUB_TOKEN`, `GITHUB_TOKEN` | `KEEL_GITHUB_OWNER` | `KEEL_GITHUB_URL` |
| `gitlab` | `KEEL_GITLAB_TOKEN`, `GITLAB_TOKEN` | `KEEL_GITLAB_OWNER` | `KEEL_GITLAB_URL` |
| `bitbucket` | `KEEL_BITBUCKET_TOKEN`, `BITBUCKET_TOKEN` | `KEEL_BITBUCKET_OWNER` | `KEEL_BITBUCKET_URL` |
| `sourcecraft` | `KEEL_SOURCECRAFT_TOKEN`, `SOURCECRAFT_TOKEN` | `KEEL_SOURCECRAFT_OWNER` | `KEEL_SOURCECRAFT_URL` |

The `KEEL_<PROVIDER>_TOKEN` form takes priority; the vendor's own variable
(`GITHUB_TOKEN`, etc.) is the fallback.

## Owner derivation

The repository owner defaults to the namespace in your `module_path`:

- `github.com/owner/repo` → `owner`
- `gitlab.com/group/subgroup/repo` → `group/subgroup` (GitLab's nested
  group/subgroup namespace is preserved)

Override it per provider with the `KEEL_<PROVIDER>_OWNER` variable from the table
above.

## Tokens never touch disk

Tokens are read from the environment at run time and used only for the API call.
They are **never written to disk** and **never recorded in `.scaffold.lock`**.

## Local-only and existing remotes

- **`provider: none`** scaffolds a purely local repository: `git init` and
  commit, but no remote is created.
- **`--remote-url <url>`** wires (or clones) an existing remote instead of
  creating a new one — useful when the repository already exists on a host. See
  [Repo states](repo-states.md) for how keel reconciles when both local and
  remote already have history.
