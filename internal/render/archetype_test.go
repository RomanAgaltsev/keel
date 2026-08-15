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

func taskfileGoLibrary(t *testing.T) string {
	t.Helper()
	return libraryPlan(t, "base-layout", "taskfile-go").Files["Taskfile.yml"]
}

func TestTaskfileGoLibraryHasNoBuildTask(t *testing.T) {
	got := taskfileGoLibrary(t)
	require.NotContains(t, got, "\n  build:")
	require.NotContains(t, got, "LDFLAGS")
	require.NotContains(t, got, "./cmd/demo")
	require.NotContains(t, got, "-X main.version=")
}

func TestTaskfileGoLibraryKeepsEveryOtherTask(t *testing.T) {
	got := taskfileGoLibrary(t)
	for _, task := range []string{
		"setup:", "formatters:install:", "golangci-lint:install:", "format:",
		"lint:", "vet:", "test:", "cover:", "deps:update:", "ci:",
		"keel:outdated:", "keel:update:", "keel:settings:",
	} {
		require.Contains(t, got, task)
	}
	// The tool pins live here regardless of archetype.
	require.Contains(t, got, "GOLANGCI_LINT_VERSION: \"v2.12.2\"")
}

func TestReadmeLibraryHasNoBuildLine(t *testing.T) {
	readme := libraryPlan(t, "base-layout").Files["README.md"]
	require.NotContains(t, readme, "task build")
	require.NotContains(t, readme, "./bin/demo")
	// The rest of Getting started survives.
	require.Contains(t, readme, "task ci")
	require.Contains(t, readme, "task         # list available tasks")
}

func TestContributingLibraryHasNoBuildLine(t *testing.T) {
	c := libraryPlan(t, "contributing-go").Files["CONTRIBUTING.md"]
	require.NotContains(t, c, "task build")
	require.Contains(t, c, "task ci")
	require.Contains(t, c, "task setup")
}

func TestReleaseGoLibraryDropsGoReleaser(t *testing.T) {
	plan := libraryPlan(t, "base-layout", "release-go")
	require.NotContains(t, plan.Files, ".goreleaser.yaml")

	wf := plan.Files[".github/workflows/release.yml"]
	require.NotContains(t, wf, "goreleaser")
	// release-please still tags and writes the changelog.
	require.Contains(t, wf, "googleapis/release-please-action@")
	require.Contains(t, plan.Files, "release-please-config.json")
	require.Contains(t, plan.Files, ".github/workflows/pr-title.yml")
}

// TestReleaseGoServiceWorkflowSurvivesTemplating is the other half: release.yml
// became a .tmpl in this task, so GitHub's ${{ }} had to be escaped. An escape
// that is wrong renders as an empty string and silently breaks releases.
func TestReleaseGoServiceWorkflowSurvivesTemplating(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "release-go"}, answers.Answers{
		"repo_name": "demo", "description": "d",
		"module_path": "github.com/acme/demo", "provider": "github",
	})
	require.NoError(t, err)

	wf := plan.Files[".github/workflows/release.yml"]
	require.Contains(t, wf, "${{ steps.rp.outputs.release_created }}")
	require.Contains(t, wf, "${{ secrets.GITHUB_TOKEN }}")
	require.Contains(t, wf, "goreleaser/goreleaser-action@")
	require.NotContains(t, wf, `{{"`) // no un-rendered escape hatch left behind
}
