package render_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2"
	"github.com/RomanAgaltsev/keel/v2/internal/answers"
	"github.com/RomanAgaltsev/keel/v2/internal/module"
	"github.com/RomanAgaltsev/keel/v2/internal/render"
)

func TestReleaseGoReleaserVerbatim(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	a := answers.Answers{"repo_name": "x", "description": "d", "module_path": "github.com/x/x", "provider": "github"}
	plan, err := render.BuildRecipe(l, []string{"base-layout", "release-go"}, a)
	require.NoError(t, err)
	require.True(t, strings.Contains(plan.Files[".goreleaser.yaml"], "{{ .Version }}"))
}

func TestReleaseGoEnforcesConventionalCommits(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "release-go"}, answers.Answers{
		"repo_name": "demo", "description": "d", "module_path": "github.com/acme/demo", "provider": "github",
	})
	require.NoError(t, err)
	wf := plan.Files[".github/workflows/pr-title.yml"]
	require.Contains(t, wf, "amannn/action-semantic-pull-request@v6")
	require.Contains(t, wf, "${{ secrets.GITHUB_TOKEN }}")
}

func TestReleaseGoBuildsTheCmdPath(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "release-go"}, answers.Answers{
		"repo_name": "demo", "description": "d", "module_path": "github.com/acme/demo", "provider": "github",
	})
	require.NoError(t, err)
	// Task 5 moved the entrypoint; GoReleaser must follow it.
	require.Contains(t, plan.Files[".goreleaser.yaml"], "./cmd/demo")
}

func TestReleaseGoPreservesGoReleaserTemplating(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "release-go"}, answers.Answers{
		"repo_name": "demo", "description": "d", "module_path": "github.com/acme/demo", "provider": "github",
	})
	require.NoError(t, err)
	got := plan.Files[".goreleaser.yaml"]
	require.Contains(t, got, "-X main.version={{ .Version }}") // GoReleaser's, not keel's
	require.Contains(t, got, "binary: demo")                   // keel's, substituted
}

// TestReleaseGoLibraryStartsAtZeroOne guards #55. Go enforces the major version
// in the import path, so a library that ships 1.0.0 and then needs a breaking
// change must move to /v2 and make every consumer edit their imports.
func TestReleaseGoLibraryStartsAtZeroOne(t *testing.T) {
	cfg := planForRecipe(t, "go-library").Files["release-please-config.json"]
	require.Contains(t, cfg, `"initial-version": "0.1.0"`)
	// Without this, release-please promotes the first breaking change straight
	// to 1.0.0 and silently undoes the choice.
	require.Contains(t, cfg, `"bump-minor-pre-major": true`)
}

func TestReleaseGoServiceStartsAtOne(t *testing.T) {
	cfg := planForRecipe(t, "go-service").Files["release-please-config.json"]
	require.Contains(t, cfg, `"initial-version": "1.0.0"`)
	require.Contains(t, cfg, `"bump-minor-pre-major": false`)
}
