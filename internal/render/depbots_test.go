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

func baseDepBotsAnswers(bot string) answers.Answers {
	return answers.Answers{
		"repo_name": "x", "description": "d", "module_path": "github.com/x/x", "dep_bot": bot,
	}
}

func TestDepBotsRenovateEnriched(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "dep-bots"}, baseDepBotsAnswers("renovate"))
	require.NoError(t, err)
	require.Contains(t, plan.Files, "renovate.json5")
	require.NotContains(t, plan.Files, ".github/dependabot.yml")
	rj := plan.Files["renovate.json5"]
	require.Contains(t, rj, "customManagers")
	require.Contains(t, rj, "golangci/golangci-lint") // the synced dep
}

func TestDepBotsDependabotHasGapNote(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "dep-bots"}, baseDepBotsAnswers("dependabot"))
	require.NoError(t, err)
	db := plan.Files[".github/dependabot.yml"]
	require.True(t, strings.Contains(db, "golangci-lint"), "must note the golangci-lint gap")
	require.True(t, strings.Contains(db, "keel outdated"), "must point at keel outdated")
}

func TestDepBotsSelect(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	a := answers.Answers{"repo_name": "x", "description": "d", "module_path": "github.com/x/x", "dep_bot": "renovate"}
	plan, err := render.BuildRecipe(l, []string{"base-layout", "dep-bots"}, a)
	require.NoError(t, err)
	require.Contains(t, plan.Files, "renovate.json5")
	require.NotContains(t, plan.Files, ".github/dependabot.yml")
}
