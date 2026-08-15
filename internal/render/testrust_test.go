package render_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2"
	"github.com/RomanAgaltsev/keel/v2/internal/answers"
	"github.com/RomanAgaltsev/keel/v2/internal/module"
	"github.com/RomanAgaltsev/keel/v2/internal/render"
)

func TestTestRustNoCoverage(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "test-rust"}, answers.Answers{
		"repo_name":      "demo",
		"description":    "a demo service",
		"module_path":    "github.com/acme/demo",
		"provider":       "github",
		"enable_codecov": false,
	})
	require.NoError(t, err)
	wf := plan.Files[".github/workflows/test.yml"]
	require.Contains(t, wf, "cargo nextest run")
	require.Contains(t, wf, "cargo test --doc")
	require.Contains(t, wf, "matrix.os")
	require.NotContains(t, plan.Files, ".github/workflows/coverage.yml") // gated off
}

func TestTestRustWithCoverage(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "test-rust"}, answers.Answers{
		"repo_name":      "demo",
		"description":    "a demo service",
		"module_path":    "github.com/acme/demo",
		"provider":       "github",
		"enable_codecov": true,
	})
	require.NoError(t, err)
	cov := plan.Files[".github/workflows/coverage.yml"]
	require.Contains(t, cov, "cargo llvm-cov")
	// Asserted by action name, not by pin: the version is owned by
	// TestTemplatePinsMatchKeelsOwnWorkflows, and hardcoding it here made this
	// test fail on every legitimate bump.
	require.Contains(t, cov, "codecov/codecov-action@")
}
