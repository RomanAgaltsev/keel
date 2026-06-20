// Package git wraps the git CLI for the operations keel needs.
package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Repo is a git working tree at Dir.
type Repo struct {
	dir string
}

// New returns a Repo handle for dir (which need not exist yet for Init).
func New(dir string) *Repo {
	return &Repo{dir: dir}
}

// Dir returns the working-tree path.
func (r *Repo) Dir() string {
	return r.dir
}

// Run executes `git -C <dir> args...` and returns combined output.
func (r *Repo) Run(ctx context.Context, args ...string) (string, error) {
	full := append([]string{"-C", r.dir}, args...)
	out, err := exec.CommandContext(ctx, "git", full...).CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// IsRepo reports whether dir already contains a git repository.
func (r *Repo) IsRepo() bool {
	_, err := os.Stat(filepath.Join(r.dir, ".git"))
	return err == nil
}

// Init initializes a repo with the given initial branch.
func (r *Repo) Init(ctx context.Context, branch string) error {
	if err := os.MkdirAll(r.dir, 0o750); err != nil {
		return err
	}
	_, err := r.Run(ctx, "init", "-b", branch)
	return err
}

// SetIdentity sets the repo-local user.name/user.email (per-repo identity convention).
func (r *Repo) SetIdentity(ctx context.Context, name, email string) error {
	if _, err := r.Run(ctx, "config", "user.name", name); err != nil {
		return err
	}
	_, err := r.Run(ctx, "config", "user.email", email)
	return err
}

// AddAll stages all changes.
func (r *Repo) AddAll(ctx context.Context) error {
	_, err := r.Run(ctx, "add", "-A")
	return err
}

// HasChanges reports whether the working tree has any staged or unstaged
// changes (so the orchestrator can skip an empty commit on idempotent re-runs).
func (r *Repo) HasChanges(ctx context.Context) (bool, error) {
	out, err := r.Run(ctx, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// Commit creates a commit with msg (identity must already be set).
func (r *Repo) Commit(ctx context.Context, msg string) error {
	_, err := r.Run(ctx, "commit", "-m", msg)
	return err
}

// AddRemote adds (or replaces) a named remote. It only falls back to set-url when
// the remote genuinely already exists; any other failure is returned as-is so its
// real cause isn't masked by a second-order set-url error.
func (r *Repo) AddRemote(ctx context.Context, name, url string) error {
	out, err := r.Run(ctx, "remote", "add", name, url)
	if err == nil {
		return nil
	}
	if !strings.Contains(out, "already exists") {
		return err
	}
	_, serr := r.Run(ctx, "remote", "set-url", name, url)
	return serr
}

// Push pushes branch to remote, setting upstream.
func (r *Repo) Push(ctx context.Context, remote, branch string) error {
	_, err := r.Run(ctx, "push", "-u", remote, branch)
	return err
}

// Clone clones url into dir and returns a Repo for it. The "--" keeps a url that
// begins with "-" from being parsed as a git flag.
func Clone(ctx context.Context, url, dir string) (*Repo, error) {
	if out, err := exec.CommandContext(ctx, "git", "clone", "--", url, dir).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git clone %q: %w: %s", url, err, strings.TrimSpace(string(out)))
	}
	return New(dir), nil
}
