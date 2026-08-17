package render_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2"
	"github.com/RomanAgaltsev/keel/v2/internal/answers"
	"github.com/RomanAgaltsev/keel/v2/internal/module"
	"github.com/RomanAgaltsev/keel/v2/internal/render"
)

func golangciConfig(t *testing.T) string {
	t.Helper()
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "lint-go"}, answers.Answers{
		"repo_name": "demo", "description": "d", "module_path": "github.com/acme/demo", "provider": "github",
	})
	require.NoError(t, err)
	return plan.Files[".golangci.yml"]
}

func TestLintGoUsesGciNotGoimports(t *testing.T) {
	got := golangciConfig(t)
	require.Contains(t, got, "gci")
	require.NotContains(t, got, "goimports") // no repo in the vault uses it
	require.Contains(t, got, "prefix(github.com/acme/demo)")
}

func TestLintGoSeedsDepguardWithOwnModule(t *testing.T) {
	got := golangciConfig(t)
	require.Contains(t, got, "$gostd")
	require.Contains(t, got, `"github.com/acme/demo"`)
}

func TestLintGoEnablesTheFullSet(t *testing.T) {
	got := golangciConfig(t)
	for _, linter := range []string{
		"gocritic", "revive", "gosec", "depguard", "bodyclose", "noctx", "cyclop",
		"gocognit", "funlen", "errorlint", "errname", "perfsprint", "prealloc",
		"forbidigo", "nolintlint", "godox", "testifylint", "sloglint",
	} {
		require.Contains(t, got, linter)
	}
}

func TestLintWorkflowDelegatesToTask(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "lint-go"}, answers.Answers{
		"repo_name": "demo", "description": "d", "module_path": "github.com/acme/demo", "provider": "github",
	})
	require.NoError(t, err)
	wf := plan.Files[".github/workflows/lint.yml"]
	require.Contains(t, wf, "task lint")
	require.Contains(t, wf, "actions/checkout@v7")
	require.NotContains(t, wf, "v2.") // no golangci version pinned here — Taskfile owns it
}

// TestLintGoRunsOnWindowsBehindAGate is #56's second half. lint ran only on
// ubuntu, which is why a CRLF working tree lint-failing on every Windows clone
// was invisible to CI for every release. Matrixing it renames the check
// context, so the gate job ships in the same change.
func TestLintGoRunsOnWindowsBehindAGate(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "lint-go"}, answers.Answers{
		"repo_name": "demo", "description": "d", "module_path": "github.com/acme/demo", "provider": "github",
	})
	require.NoError(t, err)
	wf := plan.Files[".github/workflows/lint.yml"]

	require.Contains(t, wf, "windows-latest", "lint must run on Windows or #56 stays invisible")
	require.Contains(t, wf, "lint-matrix:")
	require.Contains(t, wf, "needs: [lint-matrix]")
	require.Contains(t, wf, "if: always()")
	require.Contains(t, wf, "exit 1")
}
