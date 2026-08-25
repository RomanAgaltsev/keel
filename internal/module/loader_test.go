package module_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2"
	"github.com/RomanAgaltsev/keel/v2/internal/module"
)

func TestLoaderLoadModule(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)

	m, err := l.Load("base-layout")
	require.NoError(t, err)
	require.Equal(t, "base-layout", m.Name)

	names, err := l.ModuleNames()
	require.NoError(t, err)
	require.ElementsMatch(t, []string{
		"base-layout",
		"go-mod",
		"cli-go",
		"taskfile-go",
		"lint-go",
		"test-go",
		"security-go",
		"release-go",
		"dep-bots-go",
		"spell",
		"cargo-mod",
		"taskfile-rust",
		"lint-rust",
		"test-rust",
		"security-rust",
		"release-rust",
		"dep-bots-rust",
		"license",
		"governance",
		"contributing-go",
		"contributing-rust",
		"community-templates",
		"repo-settings-go",
		"repo-settings-rust",
	}, names)

	_, err = l.Load("does-not-exist")
	require.Error(t, err)
}

func TestLoaderTemplateFS(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)

	tfs, err := l.TemplateFS("base-layout")
	require.NoError(t, err)
	f, err := tfs.Open("README.md.tmpl")
	require.NoError(t, err)
	require.NoError(t, f.Close())
}

func TestRecipeQuestions(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)

	qs, err := module.RecipeQuestions(l, []string{"security-go"})
	require.NoError(t, err)

	ids := make([]string, len(qs))
	for i, q := range qs {
		ids[i] = q.ID
	}
	require.Contains(t, ids, "enable_codeql") // security-go contributes its own questions

	_, err = module.RecipeQuestions(l, []string{"does-not-exist"})
	require.Error(t, err)
}

func TestLoadRejectsAManifestWithABadContract(t *testing.T) {
	fsys := fstest.MapFS{
		"modules/broken/module.yaml": &fstest.MapFile{Data: []byte(
			"name: broken\nlanguage: go\ndeps:\n  - path: golang.org/x/tools\n    version: 0.38.0\n",
		)},
	}
	_, err := module.NewFSLoader(fsys).Load("broken")
	require.Error(t, err, "a manifest with an invalid deps block must not load")
	require.Contains(t, err.Error(), "v-prefixed semver")
}
