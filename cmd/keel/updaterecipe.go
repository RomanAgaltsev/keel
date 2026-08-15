package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/RomanAgaltsev/keel/v2/internal/lock"
	"github.com/RomanAgaltsev/keel/v2/internal/recipe"
)

// resolveUpdateRecipe finds the recipe an update should re-apply.
//
// Order: an explicit override, then the builtin recipe (always embedded), then
// the recorded recipe_source relative to the repo and then to the working
// directory, then a fallback that synthesizes a recipe from the lock's own
// module list. recipe_source is stored as it was typed at scaffold time —
// usually relative to where `keel new` ran, not to the repo — which is why two
// bases are tried; storing an absolute path instead would break the moment a
// repo moved between machines.
//
// The fallback returns fellBack=true and a warning rather than an error: before
// this, a moved recipe file made `keel update` fail outright (review finding
// D2), which blocks even a routine version bump. A caller that fell back must
// not treat the lock's module list as evidence that a module was removed.
func resolveUpdateRecipe(lk lock.Lock, repoPath, override string) (recipe.Recipe, string, bool, string, error) {
	if override != "" {
		rec, dir, err := loadRecipe(override)
		if err != nil {
			return recipe.Recipe{}, "", false, "", fmt.Errorf("--recipe %q: %w", override, err)
		}
		return rec, dir, false, "", nil
	}

	if lk.RecipeSource == "" {
		rec, dir, err := loadRecipe(lk.Recipe) // builtin, embedded in the binary
		if err != nil {
			return recipe.Recipe{}, "", false, "", err
		}
		return rec, dir, false, "", nil
	}

	for _, candidate := range []string{
		filepath.Join(repoPath, filepath.FromSlash(lk.RecipeSource)),
		lk.RecipeSource,
	} {
		if _, err := os.Stat(candidate); err != nil {
			continue
		}
		rec, dir, err := loadRecipe(candidate)
		if err != nil {
			return recipe.Recipe{}, "", false, "", err
		}
		return rec, dir, false, "", nil
	}

	warning := fmt.Sprintf(
		"warning: recipe %q not found; updating only the modules recorded in .scaffold.lock.\n"+
			"         Modules added to the recipe since this repo was scaffolded cannot be detected.\n"+
			"         Re-run with --recipe <path> to point at it.",
		lk.RecipeSource)
	return recipeFromLock(lk), "", true, warning, nil
}

// recipeFromLock synthesizes a recipe from what the repo already has, so an
// update can still refresh versions when the real recipe is unreachable.
func recipeFromLock(lk lock.Lock) recipe.Recipe {
	rec := recipe.Recipe{Name: lk.Recipe}
	for _, m := range lk.Modules {
		rec.Modules = append(rec.Modules, recipe.ModuleRef{Name: m.Name})
	}
	return rec
}
