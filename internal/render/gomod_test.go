package render_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2"
	"github.com/RomanAgaltsev/keel/v2/internal/answers"
	"github.com/RomanAgaltsev/keel/v2/internal/manifest"
	"github.com/RomanAgaltsev/keel/v2/internal/module"
	"github.com/RomanAgaltsev/keel/v2/internal/render"
)

// renderGoModWithDeps renders the real go-mod module alongside a synthetic
// module declaring deps, and returns the resulting go.mod.
func renderGoModWithDeps(t *testing.T, deps ...manifest.Dep) string {
	t.Helper()
	l := module.NewFSLoader(keel.BuiltinFS)
	gomod, err := l.Load("go-mod")
	require.NoError(t, err)

	p, err := render.BuildFromManifests(l,
		[]manifest.Manifest{gomod, depModule("dep-carrier", deps...)},
		answers.Answers{
			"repo_name":   "demo",
			"module_path": "github.com/RomanAgaltsev/demo",
			"archetype":   "service",
		})
	require.NoError(t, err)
	return p.Files["go.mod"]
}

func TestGoModOmitsRequireWhenNothingDeclaresADep(t *testing.T) {
	gomod := planForRecipe(t, "go-service").Files["go.mod"]
	require.NotContains(t, gomod, "require", "a stdlib-only recipe must not emit an empty require block")
	require.Equal(t, "module github.com/RomanAgaltsev/demo\n\ngo 1.26\n", gomod)
}

func TestGoModRendersDeclaredDeps(t *testing.T) {
	gomod := renderGoModWithDeps(t,
		manifest.Dep{Path: "golang.org/x/tools", Version: "v0.38.0"},
		manifest.Dep{Path: "github.com/spf13/cobra", Version: "v1.10.2"},
	)
	require.Contains(t, gomod, "require (\n")
	require.Contains(t, gomod, "\tgithub.com/spf13/cobra v1.10.2\n")
	require.Contains(t, gomod, "\tgolang.org/x/tools v0.38.0\n")
	// Sorted, so the file is stable across module resolution orders.
	require.Less(t,
		strings.Index(gomod, "github.com/spf13/cobra"),
		strings.Index(gomod, "golang.org/x/tools"))
}
