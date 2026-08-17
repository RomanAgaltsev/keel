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
//
// Use it only for assertions that do not touch the emitted union: a one-module
// plan sees only that module's own declarations, so a repo-settings module
// rendered alone correctly produces an empty check and action list. Anything
// about the union needs renderSettingsFor or renderRecipeSettings.
func renderSettings(t *testing.T, mod string, a answers.Answers) string {
	t.Helper()
	return renderSettingsFor(t, []string{mod}, a)
}

// renderSettingsFor renders an arbitrary module set, which is what the union
// tests need: the checks and actions a settings file declares come from the
// modules beside it.
func renderSettingsFor(t *testing.T, mods []string, a answers.Answers) string {
	t.Helper()
	plan, err := render.BuildRecipe(module.NewFSLoader(keel.BuiltinFS), mods, a)
	require.NoError(t, err)
	got, ok := plan.Files[settingsPath]
	require.True(t, ok, "modules %v must render %s", mods, settingsPath)
	return got
}

// goDisciplineModules is repo-settings-go plus every module that contributes a
// required check to the Go recipes.
func goDisciplineModules() []string {
	return []string{
		"base-layout", "lint-go", "test-go", "spell", "security-go", "release-go", "repo-settings-go",
	}
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
	on := goSettingsAnswers()
	on["enable_codecov"] = false
	all := renderSettingsFor(t, goDisciplineModules(), on)
	require.Contains(t, all, "- codeql")
	require.Contains(t, all, "- govulncheck")

	off := goSettingsAnswers()
	off["enable_codecov"] = false
	off["enable_codeql"] = false
	off["enable_govulncheck"] = false
	raw := renderSettingsFor(t, goDisciplineModules(), off)
	require.NotContains(t, raw, "- codeql")
	require.NotContains(t, raw, "- govulncheck")
	require.Contains(t, raw, "- lint")
	require.Contains(t, raw, "- test")
}

// The guard that used to live here, TestRepoSettingsGoChecksMatchEmittedJobNames,
// is deleted rather than updated. It compared the rendered settings file against
// a hardcoded list transcribed by eye, so both sides carried the same assumption
// and it could not detect a settings/workflow divergence by construction — and
// the list contained "test", the one context #53 proves can never report. The
// guard encoded the defect. TestRepoSettingsRendersTheEmittedUnion below, and
// the derived guards in emits_test.go, replace it.

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

// rustDisciplineModules is repo-settings-rust plus every module that
// contributes a required check to the Rust recipe.
func rustDisciplineModules() []string {
	return []string{
		"base-layout", "lint-rust", "test-rust", "spell", "security-rust", "release-rust", "repo-settings-rust",
	}
}

func rustSettingsAnswers() answers.Answers {
	return answers.Answers{
		"repo_name":          "demo",
		"description":        "a demo service",
		"module_path":        "github.com/RomanAgaltsev/demo",
		"provider":           "github",
		"visibility":         "public",
		"enable_cargo_audit": true,
		"enable_cargo_deny":  true,
		"enable_codecov":     false,
	}
}

func TestRepoSettingsRustParsesIntoSchema(t *testing.T) {
	raw := renderSettingsFor(t, rustDisciplineModules(), rustSettingsAnswers())

	var d settings.Desired
	dec := yaml.NewDecoder(strings.NewReader(raw))
	dec.KnownFields(true)
	require.NoError(t, dec.Decode(&d))
	require.NoError(t, d.Validate())
	require.Contains(t, raw, "- audit")
	require.Contains(t, raw, "- deny")
	require.NotContains(t, raw, "- coverage")
}

// TestRepoSettingsRustChecksMatchEmittedJobNames is deleted for the same reason
// as its Go twin above. Its one piece of real knowledge — that Rust names its
// dependency-review job `review`, because Rust keeps one workflow per concern
// where Go consolidates into security.yml — now lives in security-rust's own
// emits block, which is where it belongs.
func TestRepoSettingsRustChecksMatchTheEmittedUnion(t *testing.T) {
	raw, plan := renderRecipeSettings(t, "rust-service")

	var d settings.Desired
	require.NoError(t, yaml.Unmarshal([]byte(raw), &d))
	require.Len(t, d.Rulesets, 1)
	require.ElementsMatch(t, plan.Answers["emitted_checks"], d.Rulesets[0].RequiredStatusChecks)
	require.Contains(t, d.Rulesets[0].RequiredStatusChecks, "review")
}

// TestRepoSettingsRendersWithoutSecurityModule pins a failure the 2026-08-15
// acceptance run caught: enable_codeql and enable_govulncheck are declared by
// security-go, so a custom recipe taking repo-settings-go without security-go
// used to fail the whole scaffold under missingkey=error.
//
// The union makes the answer stronger than it was. Dropping the module drops
// the four checks it emits from the required list automatically, rather than
// leaving them required with no producer — which is the #53 shape, and was only
// avoided before because the answers happened to be absent too.
func TestRepoSettingsRendersWithoutSecurityModule(t *testing.T) {
	mods := []string{"base-layout", "lint-go", "test-go", "spell", "release-go", "repo-settings-go"}
	a := answers.Answers{
		"repo_name": "demo", "description": "d", "module_path": "github.com/acme/demo",
		"provider": "github", "visibility": "public", "enable_codecov": false,
		// enable_codeql / enable_govulncheck deliberately absent
	}
	raw := renderSettingsFor(t, mods, a)

	var d settings.Desired
	require.NoError(t, yaml.Unmarshal([]byte(raw), &d))
	require.Len(t, d.Rulesets, 1)
	require.ElementsMatch(t, []string{"lint", "test", "typos", "pr-title"},
		d.Rulesets[0].RequiredStatusChecks,
		"the checks security-go would have provided are dropped, not left dangling")
}

// TestRepoSettingsRustRendersWithoutOptionalAnswers checks that turning the
// optional Rust modules off drops their checks from the required list. The
// answers stay present but false rather than absent: security-rust's own file
// gates use field access, so an absent key fails its render before the settings
// template is ever reached.
func TestRepoSettingsRustRendersWithoutOptionalAnswers(t *testing.T) {
	a := rustSettingsAnswers()
	a["enable_cargo_audit"] = false
	a["enable_cargo_deny"] = false
	raw := renderSettingsFor(t, rustDisciplineModules(), a)

	var d settings.Desired
	require.NoError(t, yaml.Unmarshal([]byte(raw), &d))
	require.ElementsMatch(t, []string{
		"lint", "test", "typos", "pr-title", "review", "actionlint",
	}, d.Rulesets[0].RequiredStatusChecks)
}

// renderRecipeSettings renders a full recipe and returns its settings file with
// the plan. The one-module renderSettings helper cannot be used for anything
// touching emits: a single module's union is only its own declarations.
//
// It goes through planForRecipe so the answer set stays the one the golden
// trees use -- a second definition would drift.
func renderRecipeSettings(t *testing.T, name string) (string, render.Plan) {
	t.Helper()
	plan := planForRecipe(t, name)
	got, ok := plan.Files[settingsPath]
	require.True(t, ok, "recipe %s must render %s", name, settingsPath)
	return got, plan
}

func TestRepoSettingsRendersTheEmittedUnion(t *testing.T) {
	raw, plan := renderRecipeSettings(t, "go-library")

	var d settings.Desired
	require.NoError(t, yaml.Unmarshal([]byte(raw), &d))

	// Required checks are the union, not a hand-kept list.
	require.ElementsMatch(t, plan.Answers["emitted_checks"], d.Rulesets[0].RequiredStatusChecks)

	// Every emitted action is permitted, or its workflow gets startup_failure
	// at 0s and reports no check at all. A Docker action is referenced as
	// docker://image:tag rather than owner/repo@version, so it carries no "@*".
	for _, act := range plan.Answers["emitted_actions"].([]string) {
		want := act + "@*"
		if strings.HasPrefix(act, "docker://") {
			want = act
		}
		require.Contains(t, d.Actions.AllowedPatterns, want)
	}

	// release-go is in go-library and cannot open its release PR without this.
	require.NotNil(t, d.Actions.CanApprovePullRequestReviews)
	require.True(t, *d.Actions.CanApprovePullRequestReviews)
}
