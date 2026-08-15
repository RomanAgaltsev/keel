package render_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2"
	"github.com/RomanAgaltsev/keel/v2/internal/answers"
	"github.com/RomanAgaltsev/keel/v2/internal/module"
	"github.com/RomanAgaltsev/keel/v2/internal/render"
)

func governanceAnswers(codeOwner, provider string) answers.Answers {
	return answers.Answers{
		"repo_name": "demo", "description": "d", "module_path": "github.com/acme/demo",
		"author_name": "Ada Lovelace", "author_email": "ada@example.com",
		"code_owner": codeOwner, "provider": provider,
	}
}

func TestGovernanceEmitsAllFiles(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"governance"}, governanceAnswers("acme", "github"))
	require.NoError(t, err)
	require.Contains(t, plan.Files, "SECURITY.md")
	require.Contains(t, plan.Files, ".editorconfig")
	require.Contains(t, plan.Files, ".github/CODEOWNERS")
	require.Equal(t, "# Default owner for everything in the repo.\n* @acme\n", plan.Files[".github/CODEOWNERS"])
}

func TestGovernanceBlankCodeOwnerSkipsCodeowners(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"governance"}, governanceAnswers("", "github"))
	require.NoError(t, err)
	require.NotContains(t, plan.Files, ".github/CODEOWNERS")
	require.Contains(t, plan.Files, "SECURITY.md") // the rest still render
}

func TestGovernanceSecurityLinkIsGitHubOnly(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)

	gh, err := render.BuildRecipe(l, []string{"governance"}, governanceAnswers("acme", "github"))
	require.NoError(t, err)
	require.Contains(t, gh.Files["SECURITY.md"], "https://github.com/acme/demo/security/advisories/new")

	// Private vulnerability reporting is a GitHub feature; that URL 404s elsewhere,
	// so other providers must fall back to email.
	gl, err := render.BuildRecipe(l, []string{"governance"}, governanceAnswers("acme", "gitlab"))
	require.NoError(t, err)
	require.NotContains(t, gl.Files["SECURITY.md"], "security/advisories/new")
	require.Contains(t, gl.Files["SECURITY.md"], "ada@example.com")
}
