package render_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/render"
)

func TestSecurityCodeQLGated(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	base := answers.Answers{
		"repo_name": "x", "description": "d", "module_path": "github.com/x/x",
		"enable_codeql": false, "enable_govulncheck": true,
	}
	plan, err := render.BuildRecipe(l, []string{"base-layout", "security-go"}, base)
	require.NoError(t, err)
	require.NotContains(t, plan.Files, ".github/workflows/codeql.yml")   // gated off
	require.Contains(t, plan.Files, ".github/workflows/govulncheck.yml") // gated on
	require.Contains(t, plan.Files, ".github/workflows/dependency-review.yml")
}
