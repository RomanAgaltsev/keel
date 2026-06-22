package render_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/render"
)

func TestDepBotsRustDependabot(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "dep-bots-rust"},
		answers.Answers{"repo_name": "demo", "description": "a demo service", "dep_bot": "dependabot"})
	require.NoError(t, err)
	require.Contains(t, plan.Files[".github/dependabot.yml"], "package-ecosystem: cargo")
	require.NotContains(t, plan.Files, "renovate.json5")
}

func TestDepBotsRustRenovate(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "dep-bots-rust"},
		answers.Answers{"repo_name": "demo", "description": "a demo service", "dep_bot": "renovate"})
	require.NoError(t, err)
	require.Contains(t, plan.Files["renovate.json5"], "config:recommended")
	require.NotContains(t, plan.Files, ".github/dependabot.yml")
}
