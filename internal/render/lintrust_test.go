package render_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/render"
)

func TestLintRustRenders(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "lint-rust"}, answers.Answers{
		"repo_name":   "demo",
		"description": "a demo service",
	})
	require.NoError(t, err)

	require.Contains(t, plan.Files["rustfmt.toml"], `edition = "2024"`)
	require.Contains(t, plan.Files, "clippy.toml")
	wf := plan.Files[".github/workflows/lint.yml"]
	require.Contains(t, wf, "cargo fmt --all -- --check")
	require.Contains(t, wf, "cargo clippy --all-targets --all-features -- -D warnings")
	require.Contains(t, wf, "actions/checkout@v5")
}
