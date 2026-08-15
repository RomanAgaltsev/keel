package render_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2"
	"github.com/RomanAgaltsev/keel/v2/internal/answers"
	"github.com/RomanAgaltsev/keel/v2/internal/module"
	"github.com/RomanAgaltsev/keel/v2/internal/render"
)

func TestSecurityConsolidatedAndGated(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	base := answers.Answers{
		"repo_name": "demo", "description": "d", "module_path": "github.com/acme/demo", "provider": "github",
		"enable_codeql": false, "enable_govulncheck": true,
	}
	plan, err := render.BuildRecipe(l, []string{"base-layout", "security-go"}, base)
	require.NoError(t, err)

	// One file now, not four.
	require.Contains(t, plan.Files, ".github/workflows/security.yml")
	for _, gone := range []string{"codeql.yml", "govulncheck.yml", "dependency-review.yml", "actionlint.yml"} {
		require.NotContains(t, plan.Files, ".github/workflows/"+gone)
	}

	wf := plan.Files[".github/workflows/security.yml"]
	require.NotContains(t, wf, "codeql-action") // gated off
	require.Contains(t, wf, "govulncheck")      // gated on
	require.Contains(t, wf, "dependency-review-action@v5")
	require.Contains(t, wf, "rhysd/actionlint:1.7.7")
	// The escaped `${{` survived rendering rather than being eaten as a Go action.
	require.Contains(t, wf, "${{ github.event_name == 'pull_request' }}")
}

func TestSecurityCodeQLConfigFollowsTheJob(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	on := answers.Answers{
		"repo_name": "demo", "description": "d", "module_path": "github.com/acme/demo", "provider": "github",
		"enable_codeql": true, "enable_govulncheck": true,
	}
	plan, err := render.BuildRecipe(l, []string{"base-layout", "security-go"}, on)
	require.NoError(t, err)
	require.Contains(t, plan.Files, ".github/codeql/codeql-config.yml")
	require.Contains(t, plan.Files[".github/workflows/security.yml"], "github/codeql-action/init@v4")
}

// TestSecurityStaysValidWithEverythingGatedOff guards the shape of the `{{ if }}`
// gates: with both jobs off, dependency-review must still be the first job under
// `jobs:` rather than leaving a dangling key or a blank block.
func TestSecurityStaysValidWithEverythingGatedOff(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "security-go"}, answers.Answers{
		"repo_name": "demo", "description": "d", "module_path": "github.com/acme/demo", "provider": "github",
		"enable_codeql": false, "enable_govulncheck": false,
	})
	require.NoError(t, err)
	require.NotContains(t, plan.Files, ".github/codeql/codeql-config.yml")
	require.Contains(t, plan.Files[".github/workflows/security.yml"], "jobs:\n  dependency-review:")
}
