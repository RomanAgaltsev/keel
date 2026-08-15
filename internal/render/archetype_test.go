package render_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2"
	"github.com/RomanAgaltsev/keel/v2/internal/answers"
	"github.com/RomanAgaltsev/keel/v2/internal/module"
	"github.com/RomanAgaltsev/keel/v2/internal/render"
)

// libraryAnswers is the minimal answer set for a library render, shared by every
// archetype test.
func libraryAnswers() answers.Answers {
	return answers.Answers{
		"repo_name":   "demo",
		"description": "a demo library",
		"module_path": "github.com/acme/demo",
		"provider":    "github",
		"archetype":   "library",
	}
}

func libraryPlan(t *testing.T, modules ...string) render.Plan {
	t.Helper()
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, modules, libraryAnswers())
	require.NoError(t, err)
	return plan
}

func TestGoModLibraryEmitsRootPackage(t *testing.T) {
	plan := libraryPlan(t, "go-mod")
	require.Contains(t, plan.Files, "doc.go")
	require.Contains(t, plan.Files, "demo.go")
	require.Contains(t, plan.Files, "demo_test.go")
	require.NotContains(t, plan.Files, "cmd/demo/main.go")
	require.Contains(t, plan.Files, "go.mod")

	require.Contains(t, plan.Files["demo.go"], "package demo")
	require.Contains(t, plan.Files["demo.go"], "func Hello(name string) string")
	require.Contains(t, plan.Files["doc.go"], "// Package demo ")
	// A scaffolded repo's depguard allow-list is stdlib + its own module path.
	require.NotContains(t, plan.Files["demo_test.go"], "testify")
}

// TestRenderWithoutArchetypeIsAService pins the invariant that keeps every
// existing render call site working. Templates are parsed with
// missingkey=error, so .is_library must be present on every render — it is,
// because BuildPlan derives it, and its default is "not a library".
func TestRenderWithoutArchetypeIsAService(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"go-mod"}, answers.Answers{
		"repo_name": "demo", "description": "d",
		"module_path": "github.com/acme/demo", "provider": "github",
	})
	require.NoError(t, err)
	require.Contains(t, plan.Files, "cmd/demo/main.go")
	require.NotContains(t, plan.Files, "doc.go")
}

func TestGoModLibraryPackageNameIsSanitized(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	a := libraryAnswers()
	a["repo_name"] = "go-thing"
	plan, err := render.BuildRecipe(l, []string{"go-mod"}, a)
	require.NoError(t, err)
	require.Contains(t, plan.Files, "go-thing.go")
	require.Contains(t, plan.Files["go-thing.go"], "package gothing")
}
