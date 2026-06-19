package render_test

import (
	"flag"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/recipe"
	"github.com/RomanAgaltsev/keel/internal/render"
)

var update = flag.Bool("update", false, "update golden files")

func TestGoServiceGolden(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	rec, err := recipe.Load(keel.BuiltinFS, "go-service")
	require.NoError(t, err)

	plan, err := render.BuildRecipe(l, rec.Modules, answers.Answers{
		"repo_name":          "demo",
		"description":        "a demo service",
		"module_path":        "github.com/RomanAgaltsev/demo",
		"enable_codeql":      true,
		"enable_govulncheck": true,
		"enable_codecov":     false,
		"dep_bot":            "dependabot",
	})
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
