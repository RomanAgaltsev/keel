package render_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/render"
)

func TestReleaseRustRenders(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "release-rust"}, answers.Answers{"repo_name": "demo", "description": "a demo service"})
	require.NoError(t, err)
	require.Contains(t, plan.Files[".github/workflows/release-plz.yml"], "MarcoIeni/release-plz-action")
	require.Contains(t, plan.Files, "release-plz.toml")
}
