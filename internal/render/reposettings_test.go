package render_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/RomanAgaltsev/keel/v2"
	"github.com/RomanAgaltsev/keel/v2/internal/answers"
	"github.com/RomanAgaltsev/keel/v2/internal/module"
	"github.com/RomanAgaltsev/keel/v2/internal/render"
	"github.com/RomanAgaltsev/keel/v2/internal/settings"
)

const settingsPath = ".github/keel-settings.yml"

func goSettingsAnswers() answers.Answers {
	return answers.Answers{
		"repo_name":          "demo",
		"description":        "a demo service",
		"module_path":        "github.com/RomanAgaltsev/demo",
		"provider":           "github",
		"visibility":         "public",
		"enable_codeql":      true,
		"enable_govulncheck": true,
	}
}

// renderSettings renders one module and returns the settings file it produced.
func renderSettings(t *testing.T, mod string, a answers.Answers) string {
	t.Helper()
	plan, err := render.BuildRecipe(module.NewFSLoader(keel.BuiltinFS), []string{mod}, a)
	require.NoError(t, err)
	got, ok := plan.Files[settingsPath]
	require.True(t, ok, "module %s must render %s", mod, settingsPath)
	return got
}

// TestRepoSettingsGoParsesIntoSchema is the highest-value test here: it pins the
// template against the loader, so a typo'd key in the template fails the build
// instead of being silently ignored at apply time.
func TestRepoSettingsGoParsesIntoSchema(t *testing.T) {
	raw := renderSettings(t, "repo-settings-go", goSettingsAnswers())

	var d settings.Desired
	dec := yaml.NewDecoder(strings.NewReader(raw))
	dec.KnownFields(true)
	require.NoError(t, dec.Decode(&d))
	require.Equal(t, settings.SupportedVersion, d.Version)
	require.NoError(t, d.Validate())
}

func TestRepoSettingsGoRequiredChecksFollowAnswers(t *testing.T) {
	all := renderSettings(t, "repo-settings-go", goSettingsAnswers())
	require.Contains(t, all, "codeql")
	require.Contains(t, all, "govulncheck")

	a := goSettingsAnswers()
	a["enable_codeql"] = false
	a["enable_govulncheck"] = false
	off := renderSettings(t, "repo-settings-go", a)
	require.NotContains(t, off, "codeql")
	require.NotContains(t, off, "govulncheck")
	require.Contains(t, off, "lint")
	require.Contains(t, off, "test")
}

// TestRepoSettingsGoChecksMatchEmittedJobNames is the guard that matters most in
// this file. A required status check that no workflow ever reports blocks every
// pull request permanently — and it fails quietly, because a repo admin's own
// pushes are waved through by the bypass. The contexts here are the job names in
// the go-service golden tree, and must be updated together with them.
func TestRepoSettingsGoChecksMatchEmittedJobNames(t *testing.T) {
	var d settings.Desired
	require.NoError(t, yaml.Unmarshal([]byte(renderSettings(t, "repo-settings-go", goSettingsAnswers())), &d))
	require.Len(t, d.Rulesets, 1)
	require.ElementsMatch(t, []string{
		"lint", "test", "typos", "pr-title", "dependency-review", "actionlint", "codeql", "govulncheck",
	}, d.Rulesets[0].RequiredStatusChecks)
}

// TestRepoSettingsSecretScanningGatedOnVisibility keeps keel from writing a
// setting that is guaranteed to 422 on a private repo without Advanced Security
// ("Secret scanning is not available for this repository", verified 2026-08-15).
func TestRepoSettingsSecretScanningGatedOnVisibility(t *testing.T) {
	a := goSettingsAnswers()
	a["visibility"] = "private"
	priv := renderSettings(t, "repo-settings-go", a)
	require.Contains(t, priv, "secret_scanning: false")

	pub := renderSettings(t, "repo-settings-go", goSettingsAnswers())
	require.Contains(t, pub, "secret_scanning: true")
}

func TestRepoSettingsRustParsesIntoSchema(t *testing.T) {
	raw := renderSettings(t, "repo-settings-rust", answers.Answers{
		"repo_name":          "demo",
		"description":        "a demo service",
		"module_path":        "github.com/RomanAgaltsev/demo",
		"provider":           "github",
		"visibility":         "public",
		"enable_cargo_audit": true,
		"enable_cargo_deny":  true,
		"enable_codecov":     false,
	})
	var d settings.Desired
	dec := yaml.NewDecoder(strings.NewReader(raw))
	dec.KnownFields(true)
	require.NoError(t, dec.Decode(&d))
	require.NoError(t, d.Validate())
	require.Contains(t, raw, "audit")
	require.Contains(t, raw, "deny")
	require.NotContains(t, raw, "coverage")
}

// TestRepoSettingsRustChecksMatchEmittedJobNames is the Rust half of the guard.
// Rust keeps one workflow per concern, so its dependency-review job is named
// `review` — not `dependency-review` as in Go's consolidated security.yml.
func TestRepoSettingsRustChecksMatchEmittedJobNames(t *testing.T) {
	var d settings.Desired
	raw := renderSettings(t, "repo-settings-rust", answers.Answers{
		"repo_name": "demo", "description": "d", "module_path": "github.com/acme/demo", "provider": "github",
		"visibility": "public", "enable_cargo_audit": true, "enable_cargo_deny": true,
		"enable_codecov": true,
	})
	require.NoError(t, yaml.Unmarshal([]byte(raw), &d))
	require.Len(t, d.Rulesets, 1)
	require.ElementsMatch(t, []string{
		"lint", "test", "typos", "pr-title", "review", "actionlint", "coverage", "audit", "deny",
	}, d.Rulesets[0].RequiredStatusChecks)
}
