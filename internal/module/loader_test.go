package module_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/module"
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
