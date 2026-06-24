# Configuration

`keel config` manages a small user-level config so you don't retype your
defaults on every `keel new`. It lives at `$UserConfigDir/keel/config.yaml`
(the OS-specific user config directory).

```bash
keel config set author.name "Your Name"
keel config set author.email "you@example.com"
keel config set provider github
keel config list
```

`keel config get <key>` prints a single value; `keel config list` prints them
all. Pass `--file <path>` to operate on a config file at a specific location.

## How these become defaults

The stored `author.name`, `author.email`, and `provider` become the defaults for
`keel new`'s core questions. When a value isn't set in the config, keel falls
back to your `git config user.name` / `user.email` where applicable. You can
always override any default by answering the wizard or supplying an
[answers file](answers-and-ci.md).

## Tokens are never stored here

Provider tokens are intentionally **not** stored in the config file — they
always come from the environment. See [Providers](providers.md) for the token
environment variables.
