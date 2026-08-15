package render_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2"
	"github.com/RomanAgaltsev/keel/v2/internal/module"
	"github.com/RomanAgaltsev/keel/v2/internal/recipe"
	"github.com/RomanAgaltsev/keel/v2/internal/render"
)

func recipePlan(t *testing.T, name string) render.Plan {
	t.Helper()
	l := module.NewFSLoader(keel.BuiltinFS)
	rec, err := recipe.Load(keel.BuiltinFS, name)
	require.NoError(t, err)
	plan, err := render.BuildRecipe(l, rec.ModuleNames(), goldenAnswers(name))
	require.NoError(t, err)
	return plan
}

func TestNoStaleActionPins(t *testing.T) {
	for _, rec := range []string{"go-service", "rust-service"} {
		plan := recipePlan(t, rec)
		for path, content := range plan.Files {
			if !strings.HasPrefix(path, ".github/workflows/") {
				continue
			}
			require.NotContains(t, content, "actions/checkout@v5", "%s/%s", rec, path)
			require.NotContains(t, content, "actions/checkout@v6", "%s/%s", rec, path)
			require.NotContains(t, content, "actions/setup-go@v5", "%s/%s", rec, path)
		}
	}
}

func TestWorkflowsDoNotPinToolVersions(t *testing.T) {
	// The Taskfile owns tool versions. A workflow that pins golangci-lint too
	// is a second source of truth, and the two drifted by nine minor versions
	// before this test existed.
	for _, rec := range []string{"go-service", "rust-service"} {
		plan := recipePlan(t, rec)
		for path, content := range plan.Files {
			if !strings.HasPrefix(path, ".github/workflows/") {
				continue
			}
			require.NotContains(t, content, "golangci-lint@", "%s/%s pins a tool the Taskfile owns", rec, path)
			require.NotContains(t, content, "GOLANGCI", "%s/%s pins a tool the Taskfile owns", rec, path)
		}
	}
}

func TestTaskfileAndLintAgreeOnGolangciVersion(t *testing.T) {
	plan := recipePlan(t, "go-service")
	require.Contains(t, plan.Files["Taskfile.yml"], "v2.12.2")
}
