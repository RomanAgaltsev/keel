package render_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2"
	"github.com/RomanAgaltsev/keel/v2/internal/answers"
	"github.com/RomanAgaltsev/keel/v2/internal/module"
	"github.com/RomanAgaltsev/keel/v2/internal/render"
)

func baseLayout(t *testing.T, provider string) render.Plan {
	t.Helper()
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout"}, answers.Answers{
		"repo_name": "demo", "description": "a demo", "module_path": "github.com/acme/demo",
		"provider": provider,
	})
	require.NoError(t, err)
	return plan
}

// TestBaseLayoutShipsGitattributes guards #56. Without this file a Windows
// clone under core.autocrlf=true gets a CRLF working tree, gci and gofumpt
// normalise to LF, and `task lint` reports every Go file as unformatted --
// while CI stays green, because lint.yml runs only on ubuntu.
func TestBaseLayoutShipsGitattributes(t *testing.T) {
	got, ok := baseLayout(t, "github").Files[".gitattributes"]
	require.True(t, ok, "base-layout must render .gitattributes")
	require.Contains(t, got, "* text=auto eol=lf")
}

func TestBaseLayoutGitignoreCoversCoverage(t *testing.T) {
	got := baseLayout(t, "github").Files[".gitignore"]
	require.Contains(t, got, "coverage.out")
	require.Contains(t, got, "/bin/")
	require.Contains(t, got, "/dist/")
}

func TestBaseLayoutBadgesAreGitHubOnly(t *testing.T) {
	gh := baseLayout(t, "github").Files["README.md"]
	require.Contains(t, gh, "https://github.com/acme/demo/actions")

	// Badge URLs are GitHub-shaped; on other providers they render as broken images.
	gl := baseLayout(t, "gitlab").Files["README.md"]
	require.NotContains(t, gl, "/actions/workflows/")
	require.Contains(t, gl, "a demo") // the description still renders
}

// TestBaseLayoutBadgesNameRealWorkflows guards against badging a workflow no
// recipe emits, which renders as a permanently failing image. base-layout is
// language:any, so the badges may only name workflows both recipes produce.
func TestBaseLayoutBadgesNameRealWorkflows(t *testing.T) {
	got := baseLayout(t, "github").Files["README.md"]
	require.Contains(t, got, "workflows/lint.yml")
	require.Contains(t, got, "workflows/test.yml")
	require.NotContains(t, got, "workflows/ci.yml")
	require.NotContains(t, got, "pkg.go.dev") // Go-only, and this module is language:any
	// security-go consolidates into security.yml but security-rust does not, so a
	// security badge here would 404 in every Rust repo.
	require.NotContains(t, got, "workflows/security.yml")
}
