package update_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2/internal/lock"
	"github.com/RomanAgaltsev/keel/v2/internal/update"
)

func TestNewLockRefreshesOnlyNamedModules(t *testing.T) {
	old := lock.Lock{
		KeelVersion: "1.5.0", Recipe: "go-service",
		Modules: []lock.Module{
			{Name: "lint", Source: "builtin", Version: "1.0.0", Files: []lock.File{{Path: ".golangci.yml", SHA256: "old"}}},
			{Name: "base", Source: "builtin", Version: "1.0.0", Files: []lock.File{{Path: "README.md", SHA256: "keep"}}},
		},
		Answers: map[string]any{"repo_name": "demo"},
	}
	renderContent := map[string]string{".golangci.yml": "NEW", "README.md": "ALSO-NEW"}
	owner := map[string]string{".golangci.yml": "lint", "README.md": "base"}
	ms := update.ModuleSet{
		State:       map[string]update.State{"lint": update.Behind, "base": update.Unchanged},
		Refreshed:   map[string]string{"lint": "1.1.0"}, // only lint bumped
		Source:      map[string]string{"lint": "builtin", "base": "builtin"},
		RecipeOrder: []string{"lint", "base"},
	}

	got := update.NewLock(old, ms, renderContent, owner, "1.6.0")

	require.Equal(t, "1.6.0", got.KeelVersion)
	require.Equal(t, "go-service", got.Recipe)
	require.Equal(t, old.Answers, got.Answers)

	byName := map[string]lock.Module{}
	for _, m := range got.Modules {
		byName[m.Name] = m
	}
	// lint refreshed: version bumped, hash = HashBytes("NEW").
	require.Equal(t, "1.1.0", byName["lint"].Version)
	require.Equal(t, lock.HashBytes([]byte("NEW")), byName["lint"].Files[0].SHA256)
	// base untouched: old entry preserved verbatim.
	require.Equal(t, "1.0.0", byName["base"].Version)
	require.Equal(t, "keep", byName["base"].Files[0].SHA256)
}

func TestNewLockAppendsAddedModules(t *testing.T) {
	old := lock.Lock{Modules: []lock.Module{
		{Name: "base-layout", Source: "builtin", Version: "1.0.0"},
	}}
	ms := update.ModuleSet{
		State:       map[string]update.State{"base-layout": update.Unchanged, "license": update.Added},
		Refreshed:   map[string]string{"license": "1.0.0"},
		Source:      map[string]string{"license": "builtin"},
		RecipeOrder: []string{"base-layout", "license"},
	}
	got := update.NewLock(old, ms,
		map[string]string{"LICENSE": "MIT License\n"},
		map[string]string{"LICENSE": "license"},
		"2.1.0")

	require.Len(t, got.Modules, 2)
	require.Equal(t, "license", got.Modules[1].Name)
	require.Equal(t, "1.0.0", got.Modules[1].Version)
	require.Equal(t, "builtin", got.Modules[1].Source)
	require.Equal(t, []lock.File{{
		Path:   "LICENSE",
		SHA256: lock.HashBytes([]byte("MIT License\n")),
	}}, got.Modules[1].Files)
}

func TestNewLockDropsOrphanedModules(t *testing.T) {
	old := lock.Lock{Modules: []lock.Module{
		{Name: "base-layout", Source: "builtin", Version: "1.0.0"},
		{Name: "spell", Source: "builtin", Version: "1.0.0"},
	}}
	ms := update.ModuleSet{
		State:       map[string]update.State{"base-layout": update.Unchanged, "spell": update.Orphaned},
		RecipeOrder: []string{"base-layout"},
	}
	got := update.NewLock(old, ms, map[string]string{}, map[string]string{}, "2.1.0")

	require.Len(t, got.Modules, 1)
	require.Equal(t, "base-layout", got.Modules[0].Name)
}

func TestNewLockPreservesRecipeSource(t *testing.T) {
	old := lock.Lock{Recipe: "my-recipe", RecipeSource: "./my-recipe.yaml"}
	got := update.NewLock(old, update.ModuleSet{State: map[string]update.State{}},
		map[string]string{}, map[string]string{}, "2.1.0")
	require.Equal(t, "./my-recipe.yaml", got.RecipeSource)
}
