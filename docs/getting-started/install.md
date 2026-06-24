# Install

Install `keel` with the Go toolchain:

```bash
go install github.com/RomanAgaltsev/keel/cmd/keel@latest
```

This drops a `keel` binary into `$(go env GOPATH)/bin` — make sure that
directory is on your `PATH`.

Prefer a prebuilt binary? Download one for your platform from the
[releases page](https://github.com/RomanAgaltsev/keel/releases) and put it
somewhere on your `PATH`.

Building from source requires **Go 1.26+**. (You don't need Go to run a
released binary — only to `go install` or build the project yourself.)

## Confirm the install

```bash
keel version
```

This prints the version, commit, and build date, e.g.
`keel 1.7.1 (commit abc1234, built 2026-06-23)`. If you see a version string,
you're ready for the [Quick start](quickstart.md).
