package render_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/render"
)

func TestReleaseGoReleaserVerbatim(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	a := answers.Answers{"repo_name": "x", "description": "d", "module_path": "github.com/x/x"}
	plan, err := render.BuildRecipe(l, []string{"base-layout", "release-go"}, a)
	require.NoError(t, err)
	require.True(t, strings.Contains(plan.Files[".goreleaser.yaml"], "{{ .Version }}"))
}

func TestReleaseGoEnforcesConventionalCommits(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "release-go"}, answers.Answers{
		"repo_name": "demo", "description": "d", "module_path": "github.com/acme/demo",
	})
	require.NoError(t, err)
	wf := plan.Files[".github/workflows/pr-title.yml"]
	require.Contains(t, wf, "amannn/action-semantic-pull-request@v6")
	require.Contains(t, wf, "${{ secrets.GITHUB_TOKEN }}")
}

func TestReleaseGoBuildsTheCmdPath(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "release-go"}, answers.Answers{
		"repo_name": "demo", "description": "d", "module_path": "github.com/acme/demo",
	})
	require.NoError(t, err)
	// Task 5 moved the entrypoint; GoReleaser must follow it.
	require.Contains(t, plan.Files[".goreleaser.yaml"], "./cmd/demo")
}

func TestReleaseGoPreservesGoReleaserTemplating(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "release-go"}, answers.Answers{
		"repo_name": "demo", "description": "d", "module_path": "github.com/acme/demo",
	})
	require.NoError(t, err)
	got := plan.Files[".goreleaser.yaml"]
	require.Contains(t, got, "-X main.version={{ .Version }}") // GoReleaser's, not keel's
	require.Contains(t, got, "binary: demo")                   // keel's, substituted
}
