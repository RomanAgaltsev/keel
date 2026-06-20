// Package source resolves external module sources (local dir or git) into an
// fs.FS rooted at the module directory, plus provenance for the lockfile.
package source

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

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

func resolveGit(_ context.Context, _ recipe.Source) (Resolved, error) {
	return Resolved{}, fmt.Errorf("git sources not yet implemented")
}
