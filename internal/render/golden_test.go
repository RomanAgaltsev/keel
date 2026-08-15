package render_test

import (
	"flag"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2"
	"github.com/RomanAgaltsev/keel/v2/internal/answers"
	"github.com/RomanAgaltsev/keel/v2/internal/module"
	"github.com/RomanAgaltsev/keel/v2/internal/recipe"
	"github.com/RomanAgaltsev/keel/v2/internal/render"
)

var update = flag.Bool("update", false, "update golden files")

// goldenAnswers is the single definition of the answer set the golden trees are
// rendered from. Every test that builds a whole recipe uses it, so a new answer
// is added in one place rather than drifting between the two golden tests.
func goldenAnswers(recipeName string) answers.Answers {
	a := answers.Answers{
		"repo_name":      "demo",
		"description":    "a demo service",
		"module_path":    "github.com/RomanAgaltsev/demo",
		"author_name":    "Roman Agaltsev",
		"author_email":   "roman-agalcev@yandex.ru",
		"license":        "MIT",
		"year":           2026,
		"provider":       "github",
		"visibility":     "public",
		"code_owner":     "RomanAgaltsev",
		"enable_codecov": true,
		"dep_bot":        "dependabot",
	}
	switch recipeName {
	case "go-service":
		a["enable_codeql"] = true
		a["enable_govulncheck"] = true
	case "go-library":
		a["description"] = "a demo library"
		a["enable_codeql"] = true
		a["enable_govulncheck"] = true
	case "rust-service":
		a["enable_cargo_audit"] = true
		a["enable_cargo_deny"] = true
	}
	return a
}

// planForRecipe renders a builtin recipe the way `keel new` does: the archetype
// is taken from the recipe, not from the answer file.
func planForRecipe(t *testing.T, name string) render.Plan {
	t.Helper()
	l := module.NewFSLoader(keel.BuiltinFS)
	rec, err := recipe.Load(keel.BuiltinFS, name)
	require.NoError(t, err)
	a := goldenAnswers(name)
	a["archetype"] = rec.Archetype
	plan, err := render.BuildRecipe(l, rec.ModuleNames(), a)
	require.NoError(t, err)
	return plan
}

// assertGolden compares a rendered plan against a golden tree, or rewrites the
// tree when -update is passed.
func assertGolden(t *testing.T, plan render.Plan, recipeName string) {
	t.Helper()
	goldenDir := filepath.Join("testdata", "golden", recipeName)
	if *update {
		require.NoError(t, os.RemoveAll(goldenDir))
		require.NoError(t, render.WritePlan(plan, goldenDir))
		return
	}

	want := map[string]string{}
	require.NoError(t, filepath.WalkDir(goldenDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(goldenDir, p)
		want[filepath.ToSlash(rel)] = string(b)
		return nil
	}))

	require.Equal(t, keys(want), keys(plan.Files), "file set differs from golden")
	for k, v := range want {
		require.Equal(t, v, plan.Files[k], "content differs for %s", k)
	}
}

func TestGoServiceGolden(t *testing.T) {
	assertGolden(t, planForRecipe(t, "go-service"), "go-service")
}

func TestGoLibraryGolden(t *testing.T) {
	assertGolden(t, planForRecipe(t, "go-library"), "go-library")
}

// TestGoLibraryGoldenHasNoBinaryMachinery states the point of the recipe
// directly, so a future regression names itself instead of showing up as an
// opaque golden diff.
func TestGoLibraryGoldenHasNoBinaryMachinery(t *testing.T) {
	plan := planForRecipe(t, "go-library")
	require.NotContains(t, plan.Files, ".goreleaser.yaml")
	for dest := range plan.Files {
		require.NotContains(t, dest, "cmd/")
	}
	require.Contains(t, plan.Files, "doc.go")
	require.Contains(t, plan.Files, "demo.go")
	require.NotContains(t, plan.Files["Taskfile.yml"], "\n  build:")
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
