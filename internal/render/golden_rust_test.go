package render_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/recipe"
	"github.com/RomanAgaltsev/keel/internal/render"
)

func TestRustServiceGolden(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	rec, err := recipe.Load(keel.BuiltinFS, "rust-service")
	require.NoError(t, err)

	plan, err := render.BuildRecipe(l, rec.ModuleNames(), answers.Answers{
		"repo_name":          "demo",
		"description":        "a demo service",
		"module_path":        "github.com/RomanAgaltsev/demo",
		"author_name":        "Roman Agaltsev",
		"author_email":       "roman-agalcev@yandex.ru",
		"license":            "MIT",
		"enable_codecov":     false,
		"enable_cargo_audit": true,
		"enable_cargo_deny":  true,
		"dep_bot":            "dependabot",
	})

	require.NoError(t, err)

	goldenDir := filepath.Join("testdata", "golden", "rust-service")
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
