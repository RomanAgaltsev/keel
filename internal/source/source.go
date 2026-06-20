// Package source resolves external module sources (local dir or git) into an
// fs.FS rooted at the module directory, plus provenance for the lockfile.
package source

import (
	"context"
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
		return Resolved{}, fmt.Errorf("source has neither dir nor git")
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
	if _, statErr := os.Stat(dir); statErr != nil {
		if err := clone(ctx, src.Git, src.Ref, dir); err != nil {
			_ = os.RemoveAll(dir)
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
// falls back to a full clone + checkout.
func clone(ctx context.Context, url, ref, dir string) error {
	if err := os.MkdirAll(filepath.Dir(dir), 0o750); err != nil {
		return err
	}
	if out, err := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--branch", ref, url, dir).CombinedOutput(); err == nil {
		return nil
	} else {
		_ = out
	}
	_ = os.RemoveAll(dir)
	if out, err := exec.CommandContext(ctx, "git", "clone", url, dir).CombinedOutput(); err != nil {
		return fmt.Errorf("git clone %s: %w: %s", url, err, strings.TrimSpace(string(out)))
	}
	if out, err := exec.CommandContext(ctx, "git", "-C", dir, "checkout", ref).CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout %s in %s: %w: %s", ref, url, err, strings.TrimSpace(string(out)))
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
