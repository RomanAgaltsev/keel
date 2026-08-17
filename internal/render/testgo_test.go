package render_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2"
	"github.com/RomanAgaltsev/keel/v2/internal/answers"
	"github.com/RomanAgaltsev/keel/v2/internal/module"
	"github.com/RomanAgaltsev/keel/v2/internal/render"
)

func testGoPlan(t *testing.T, codecov bool) render.Plan {
	t.Helper()
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "test-go"}, answers.Answers{
		"repo_name": "demo", "description": "d", "module_path": "github.com/acme/demo", "provider": "github",
		"enable_codecov": codecov,
	})
	require.NoError(t, err)
	return plan
}

func TestTestGoCodecovEnabled(t *testing.T) {
	plan := testGoPlan(t, true)
	require.Contains(t, plan.Files, "codecov.yml")
	require.Contains(t, plan.Files["codecov.yml"], "cmd/demo/main.go")
	wf := plan.Files[".github/workflows/test.yml"]
	require.Contains(t, wf, "codecov/codecov-action@")
	require.Contains(t, wf, "${{ secrets.CODECOV_TOKEN }}")
}

func TestTestGoCodecovDisabled(t *testing.T) {
	plan := testGoPlan(t, false)
	require.NotContains(t, plan.Files, "codecov.yml")
	require.NotContains(t, plan.Files[".github/workflows/test.yml"], "codecov")
}

func TestTestGoMatrixDoesNotFailFast(t *testing.T) {
	wf := testGoPlan(t, true).Files[".github/workflows/test.yml"]
	require.Contains(t, wf, "fail-fast: false")
	require.Contains(t, wf, "${{ matrix.os }}")
	require.Contains(t, wf, "actions/checkout@v7")
}

// TestTestGoEmitsAStableCheckContext guards #53. A matrixed job reports
// `test (ubuntu-latest)` and never a bare `test`, so a ruleset requiring `test`
// waits on a producer that does not exist and every PR sits BLOCKED with all
// runs green.
func TestTestGoEmitsAStableCheckContext(t *testing.T) {
	wf := testGoPlan(t, false).Files[".github/workflows/test.yml"]

	require.Contains(t, wf, "test-matrix:", "the matrixed job must be renamed")
	require.Contains(t, wf, "needs: [test-matrix]", "the gate must depend on the matrix")
	require.Contains(t, wf, "if: always()",
		"without always() the gate is skipped when the matrix fails, and a skipped required check does not block")
	require.Contains(t, wf, "exit 1",
		"the gate must fail explicitly; reporting success by not running is the defect it prevents")
}
