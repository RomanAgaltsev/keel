// Package git wraps the git CLI for the operations keel needs.
package git

import (
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
func (r *Repo) Run(args ...string) (string, error) {
	full := append([]string{"-C", r.dir}, args...)
	out, err := exec.Command("git", full...).CombinedOutput()
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
func (r *Repo) Init(branch string) error {
	if err := os.MkdirAll(r.dir, 0o750); err != nil {
		return err
	}
	_, err := r.Run("init", "-b", branch)
	return err
}

// SetIdentity sets the repo-local user.name/user.email (per-repo identity convention).
func (r *Repo) SetIdentity(name, email string) error {
	if _, err := r.Run("config", "user.name", name); err != nil {
		return err
	}
	_, err := r.Run("config", "user.email", email)
	return err
}

// AddAll stages all changes.
func (r *Repo) AddAll() error {
	_, err := r.Run("add", "-A")
	return err
}

// HasChanges reports whether the working tree has any staged or unstaged
// changes (so the orchestrator can skip an empty commit on idempotent re-runs).
func (r *Repo) HasChanges() (bool, error) {
	out, err := r.Run("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// Commit creates a commit with msg (identity must already be set).
func (r *Repo) Commit(msg string) error {
	_, err := r.Run("commit", "-m", msg)
	return err
}

// AddRemote adds (or replaces) a named remote.
func (r *Repo) AddRemote(name, url string) error {
	if _, err := r.Run("remote", "add", name, url); err != nil {
		// If it already exists, set the URL instead.
		_, serr := r.Run("remote", "set-url", name, url)
		return serr
	}
	return nil
}

// Push pushes branch to remote, setting upstream.
func (r *Repo) Push(remote, branch string) error {
	_, err := r.Run("push", "-u", remote, branch)
	return err
}

// Clone clones url into dir and returns a Repo for it.
func Clone(url, dir string) (*Repo, error) {
	if _, err := exec.Command("git", "clone", url, dir).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git clone %q: %w", url, err)
	}
	return New(dir), nil
}
