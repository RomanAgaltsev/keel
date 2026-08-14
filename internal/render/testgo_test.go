package render_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/render"
)

func testGoPlan(t *testing.T, codecov bool) render.Plan {
	t.Helper()
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "test-go"}, answers.Answers{
		"repo_name": "demo", "description": "d", "module_path": "github.com/acme/demo",
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
	require.Contains(t, wf, "codecov/codecov-action@v7")
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
