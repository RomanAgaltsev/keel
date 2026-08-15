package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/lock"
)

func writeRecipe(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(
		"name: my-recipe\nlanguage: go\nmodules: [base-layout, go-mod]\n"), 0o600))
}

func TestResolveUpdateRecipeBuiltin(t *testing.T) {
	rec, _, fellBack, warn, err := resolveUpdateRecipe(
		lock.Lock{Recipe: "go-service"}, t.TempDir(), "")
	require.NoError(t, err)
	require.False(t, fellBack)
	require.Empty(t, warn)
	require.Equal(t, "go-service", rec.Name)
	require.Contains(t, rec.ModuleNames(), "license")
}

func TestResolveUpdateRecipeFileRelativeToRepo(t *testing.T) {
	repo := t.TempDir()
	writeRecipe(t, filepath.Join(repo, "recipes", "my-recipe.yaml"))

	rec, _, fellBack, _, err := resolveUpdateRecipe(
		lock.Lock{Recipe: "my-recipe", RecipeSource: "recipes/my-recipe.yaml"}, repo, "")
	require.NoError(t, err)
	require.False(t, fellBack)
	require.Equal(t, []string{"base-layout", "go-mod"}, rec.ModuleNames())
}

func TestResolveUpdateRecipeOverrideWins(t *testing.T) {
	repo := t.TempDir()
	other := filepath.Join(t.TempDir(), "other.yaml")
	writeRecipe(t, other)

	rec, _, fellBack, _, err := resolveUpdateRecipe(
		lock.Lock{Recipe: "my-recipe", RecipeSource: "gone.yaml"}, repo, other)
	require.NoError(t, err)
	require.False(t, fellBack)
	require.Equal(t, []string{"base-layout", "go-mod"}, rec.ModuleNames())
}

func TestResolveUpdateRecipeFallsBackToLock(t *testing.T) {
	// D2 used to be a hard error here. Falling back lets a routine version-bump
	// update proceed when the recipe file has merely moved.
	lk := lock.Lock{
		Recipe:       "my-recipe",
		RecipeSource: "recipes/gone.yaml",
		Modules: []lock.Module{
			{Name: "base-layout", Source: "builtin", Version: "1.0.0"},
			{Name: "go-mod", Source: "builtin", Version: "1.0.0"},
		},
	}
	rec, _, fellBack, warn, err := resolveUpdateRecipe(lk, t.TempDir(), "")
	require.NoError(t, err)
	require.True(t, fellBack)
	require.Contains(t, warn, "recipes/gone.yaml")
	require.Contains(t, warn, "--recipe")
	require.Equal(t, []string{"base-layout", "go-mod"}, rec.ModuleNames())
}

func TestResolveUpdateRecipeOverrideMissingIsAnError(t *testing.T) {
	// An explicit --recipe that does not resolve is a typo, not a reason to
	// silently update something else.
	_, _, _, _, err := resolveUpdateRecipe(
		lock.Lock{Recipe: "my-recipe"}, t.TempDir(), "no-such-file.yaml")
	require.Error(t, err)
}
