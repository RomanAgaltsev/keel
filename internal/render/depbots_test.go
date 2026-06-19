package render_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/render"
)

func TestDepBotsSelect(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	a := answers.Answers{"repo_name": "x", "description": "d", "module_path": "github.com/x/x", "dep_bot": "renovate"}
	plan, err := render.BuildRecipe(l, []string{"base-layout", "dep-bots"}, a)
	require.NoError(t, err)
	require.Contains(t, plan.Files, "renovate.json5")
	require.NotContains(t, plan.Files, ".github/dependabot.yml")
}
