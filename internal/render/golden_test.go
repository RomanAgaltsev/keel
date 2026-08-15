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
	case "rust-service":
		a["enable_cargo_audit"] = true
		a["enable_cargo_deny"] = true
	}
	return a
}

func TestGoServiceGolden(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	rec, err := recipe.Load(keel.BuiltinFS, "go-service")
	require.NoError(t, err)

	plan, err := render.BuildRecipe(l, rec.ModuleNames(), goldenAnswers("go-service"))
	require.NoError(t, err)

	goldenDir := filepath.Join("testdata", "golden", "go-service")
	if *update {
		require.NoError(t, os.RemoveAll(goldenDir))
		require.NoError(t, render.WritePlan(plan, goldenDir))
		return
	}

	// Compare plan against the golden tree.
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

	gotKeys := keys(plan.Files)
	wantKeys := keys(want)
	require.Equal(t, wantKeys, gotKeys, "file set differs from golden")
	for k, v := range want {
		require.Equal(t, v, plan.Files[k], "content differs for %s", k)
	}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
