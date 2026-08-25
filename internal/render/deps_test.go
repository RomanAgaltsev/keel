package render_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2/internal/answers"
	"github.com/RomanAgaltsev/keel/v2/internal/manifest"
	"github.com/RomanAgaltsev/keel/v2/internal/module"
	"github.com/RomanAgaltsev/keel/v2/internal/render"
)

// depModule returns a manifest declaring the given deps and rendering no files,
// so a test can exercise the union without a template tree.
func depModule(name string, deps ...manifest.Dep) manifest.Manifest {
	return manifest.Manifest{Name: name, Language: "go", Deps: deps}
}

// buildDepPlan builds a plan from the given manifests and returns its dep union.
// An empty MapFS suffices as the loader: none of these modules renders a file.
func buildDepPlan(t *testing.T, mans ...manifest.Manifest) []manifest.Dep {
	t.Helper()
	l := module.NewFSLoader(fstest.MapFS{})
	p, err := render.BuildFromManifests(l, mans, answers.Answers{
		"repo_name":   "demo",
		"module_path": "github.com/RomanAgaltsev/demo",
	})
	require.NoError(t, err)
	deps, ok := p.Answers["go_deps"].([]manifest.Dep)
	require.True(t, ok, "go_deps must be a []manifest.Dep")
	return deps
}

func TestGoDepsIsAlwaysSetEvenWhenEmpty(t *testing.T) {
	// missingkey=error means a template naming .go_deps must not explode on a
	// recipe where nothing declares a dependency.
	plan := planForRecipe(t, "go-service")
	got, ok := plan.Answers["go_deps"]
	require.True(t, ok, "go_deps must always be set")
	require.Empty(t, got.([]manifest.Dep))
}

func TestUnionDepsIsSortedAndOrderIndependent(t *testing.T) {
	a := depModule("mod-a", manifest.Dep{Path: "golang.org/x/tools", Version: "v0.38.0"})
	b := depModule("mod-b", manifest.Dep{Path: "github.com/spf13/cobra", Version: "v1.10.2"})

	forward := buildDepPlan(t, a, b)
	reverse := buildDepPlan(t, b, a)

	want := []manifest.Dep{
		{Path: "github.com/spf13/cobra", Version: "v1.10.2"},
		{Path: "golang.org/x/tools", Version: "v0.38.0"},
	}
	require.Equal(t, want, forward, "sorted by path")
	require.Equal(t, forward, reverse, "module resolution order must not change the union")
}

func TestUnionDepsTakesTheHighestVersion(t *testing.T) {
	lo := depModule("mod-lo", manifest.Dep{Path: "golang.org/x/tools", Version: "v0.38.0"})
	hi := depModule("mod-hi", manifest.Dep{Path: "golang.org/x/tools", Version: "v0.39.1"})

	require.Equal(t,
		[]manifest.Dep{{Path: "golang.org/x/tools", Version: "v0.39.1"}},
		buildDepPlan(t, lo, hi))
	require.Equal(t,
		[]manifest.Dep{{Path: "golang.org/x/tools", Version: "v0.39.1"}},
		buildDepPlan(t, hi, lo),
		"highest wins regardless of declaration order")
}
