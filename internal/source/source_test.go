package source_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/recipe"
	"github.com/RomanAgaltsev/keel/internal/source"
)

func writeModuleDir(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "templates"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "module.yaml"),
		[]byte("name: logging\nversion: 1.2.0\nlanguage: go\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "templates", "log.go.tmpl"),
		[]byte("package log\n"), 0o644))
}

func TestResolveDir(t *testing.T) {
	root := t.TempDir()
	modDir := filepath.Join(root, "mods", "logging")
	writeModuleDir(t, modDir)

	// recipeDir is root; the recipe references ./mods/logging relative to it.
	res, err := source.Resolve(context.Background(), recipe.Source{Dir: "./mods/logging"}, root)
	require.NoError(t, err)
	require.Equal(t, "dir:./mods/logging", res.Source)
	require.Equal(t, "1.2.0", res.Version)

	b, err := fs.ReadFile(res.FS, "module.yaml")
	require.NoError(t, err)
	require.Contains(t, string(b), "name: logging")
}

func TestResolveDirMissing(t *testing.T) {
	_, err := source.Resolve(context.Background(), recipe.Source{Dir: "./nope"}, t.TempDir())
	require.Error(t, err)
}
