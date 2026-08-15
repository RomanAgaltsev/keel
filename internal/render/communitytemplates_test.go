package render_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2"
	"github.com/RomanAgaltsev/keel/v2/internal/answers"
	"github.com/RomanAgaltsev/keel/v2/internal/module"
	"github.com/RomanAgaltsev/keel/v2/internal/render"
)

func communityAnswers(provider string) answers.Answers {
	return answers.Answers{
		"repo_name": "demo", "description": "d", "module_path": "github.com/acme/demo",
		"author_email": "ada@example.com", "provider": provider,
	}
}

func TestCommunityTemplatesEmitted(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"community-templates"}, communityAnswers("github"))
	require.NoError(t, err)
	require.Contains(t, plan.Files, ".github/ISSUE_TEMPLATE/bug_report.yml")
	require.Contains(t, plan.Files, ".github/ISSUE_TEMPLATE/feature_request.yml")
	require.Contains(t, plan.Files, ".github/ISSUE_TEMPLATE/config.yml")
	require.Contains(t, plan.Files, ".github/PULL_REQUEST_TEMPLATE.md")
	require.Contains(t, plan.Files[".github/ISSUE_TEMPLATE/bug_report.yml"], "Report a problem with demo")
	require.Contains(t, plan.Files[".github/ISSUE_TEMPLATE/config.yml"], "blank_issues_enabled: false")
}

func TestCommunityBugReportIsLanguageNeutral(t *testing.T) {
	// This module is language:any and lands in rust-service too, so the form must
	// not ask for a Go version.
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"community-templates"}, communityAnswers("github"))
	require.NoError(t, err)
	require.NotContains(t, plan.Files[".github/ISSUE_TEMPLATE/bug_report.yml"], "Go version")
	require.Contains(t, plan.Files[".github/ISSUE_TEMPLATE/bug_report.yml"], "Environment")
}

func TestCommunityConfigSecurityLinkIsGitHubOnly(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	gl, err := render.BuildRecipe(l, []string{"community-templates"}, communityAnswers("gitlab"))
	require.NoError(t, err)
	require.NotContains(t, gl.Files[".github/ISSUE_TEMPLATE/config.yml"], "security/advisories/new")
}
