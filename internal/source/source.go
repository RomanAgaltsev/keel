// Package source resolves external module sources (local dir or git) into an
// fs.FS rooted at the module directory, plus provenance for the lockfile.
package source

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/RomanAgaltsev/keel/internal/recipe"
)

// Resolved is a located external module ready to load.
type Resolved struct {
	FS      fs.FS  // rooted at the module dir (module.yaml + templates/ at its root)
	Source  string // "dir:<path>" or "git:<url>//<subdir>@<ref>"
	Version string // module.yaml version (dir) or commit SHA (git)
}

// Resolve turns a recipe Source into a Resolved module. recipeDir is the directory
// of the recipe file, used to resolve relative dir sources.
func Resolve(ctx context.Context, src recipe.Source, recipeDir string) (Resolved, error) {
	switch {
	case src.Dir != "":
		return resolveDir(src.Dir, recipeDir)
	case src.Git != "":
		return resolveGit(ctx, src)
	default:
		return Resolved{}, errors.New("source has neither dir nor git")
	}
}

func resolveDir(dir, recipeDir string) (Resolved, error) {
	abs := dir
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(recipeDir, dir)
	}
	fsys := os.DirFS(abs)
	ver, err := manifestVersion(fsys)
	if err != nil {
		return Resolved{}, fmt.Errorf("dir source %q: %w", dir, err)
	}
	return Resolved{FS: fsys, Source: "dir:" + dir, Version: ver}, nil
}

func manifestVersion(fsys fs.FS) (string, error) {
	b, err := fs.ReadFile(fsys, "module.yaml")
	if err != nil {
		return "", fmt.Errorf("module.yaml not found: %w", err)
	}
	var doc struct {
		Version string `yaml:"version"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return "", err
	}
	return doc.Version, nil
}

func resolveGit(ctx context.Context, src recipe.Source) (Resolved, error) {
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return Resolved{}, err
	}
	dir := filepath.Join(cacheRoot, "keel", "modules", slug(src.Git)+"@"+sanitize(src.Ref))

	switch {
	case !cacheValid(dir):
		// No usable cache (absent, or a partial/aborted clone). Start clean.
		_ = os.RemoveAll(dir) //nolint:gosec // drop any partial cache before a fresh clone
		if err := clone(ctx, src.Git, src.Ref, dir); err != nil {
			_ = os.RemoveAll(dir) //nolint:gosec // best-effort cleanup of a partial clone; the clone error is returned
			return Resolved{}, err
		}
	case isMutableRef(src.Ref):
		// A branch or tag can move upstream, so a cached clone may be stale; refresh
		// it to the current tip. A full commit SHA is immutable — its cache is
		// authoritative and left untouched (no network).
		if err := fetchReset(ctx, dir, src.Ref); err != nil {
			return Resolved{}, err
		}
	}

	sha, err := headSHA(ctx, dir)
	if err != nil {
		return Resolved{}, err
	}
	modRoot := dir
	if src.Subdir != "" {
		modRoot = filepath.Join(dir, filepath.FromSlash(src.Subdir))
	}
	fsys := os.DirFS(modRoot)
	if _, err := fs.Stat(fsys, "module.yaml"); err != nil {
		return Resolved{}, fmt.Errorf("module dir %q not found in %s@%s", src.Subdir, src.Git, src.Ref)
	}
	return Resolved{
		FS:      fsys,
		Source:  fmt.Sprintf("git:%s//%s@%s", src.Git, src.Subdir, src.Ref),
		Version: sha,
	}, nil
}

// clone shallow-clones ref; if ref is a commit SHA (which --branch rejects), it
// falls back to a full clone + checkout. The "--" end-of-options separators keep a
// ref or url that begins with "-" from being parsed by git as a flag.
func clone(ctx context.Context, url, ref, dir string) error {
	if err := os.MkdirAll(filepath.Dir(dir), 0o750); err != nil {
		return err
	}
	// Try a fast shallow clone of the ref first; on failure fall back below.
	if _, err := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--branch", ref, "--", url, dir).CombinedOutput(); err == nil {
		return nil
	}
	_ = os.RemoveAll(dir) //nolint:gosec // discard the partial shallow clone before retrying with a full clone
	if out, err := exec.CommandContext(ctx, "git", "clone", "--", url, dir).CombinedOutput(); err != nil {
		return fmt.Errorf("git clone %s: %w: %s", url, err, strings.TrimSpace(string(out)))
	}
	if out, err := exec.CommandContext(ctx, "git", "-C", dir, "checkout", ref, "--").CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout %s in %s: %w: %s", ref, url, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// cacheValid reports whether dir holds a usable clone (a populated .git), so a
// partial or aborted clone left behind by a killed process is not trusted.
func cacheValid(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

// isMutableRef reports whether ref can move upstream. Only a full 40-character hex
// commit SHA is treated as immutable; branches and tags are refreshed on resolve.
func isMutableRef(ref string) bool {
	if len(ref) != 40 {
		return true
	}
	for _, r := range ref {
		hex := (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
		if !hex {
			return true
		}
	}
	return false
}

// fetchReset updates an existing cache clone to the upstream tip of ref.
func fetchReset(ctx context.Context, dir, ref string) error {
	if out, err := exec.CommandContext(ctx, "git", "-C", dir, "fetch", "--depth", "1", "origin", ref).CombinedOutput(); err != nil {
		return fmt.Errorf("git fetch %s in %s: %w: %s", ref, dir, err, strings.TrimSpace(string(out)))
	}
	if out, err := exec.CommandContext(ctx, "git", "-C", dir, "reset", "--hard", "FETCH_HEAD").CombinedOutput(); err != nil {
		return fmt.Errorf("git reset in %s: %w: %s", dir, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func headSHA(ctx context.Context, dir string) (string, error) {
	out, err := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("rev-parse HEAD in %s: %w", dir, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// slug turns a repo URL into a path-safe cache key fragment.
func slug(url string) string {
	s := strings.TrimSuffix(url, ".git")
	s = strings.ReplaceAll(s, "://", "/")
	return sanitize(s)
}

func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '/', '.', '-', '_':
			return r
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, s)
}
