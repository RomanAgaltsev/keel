package manifest_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/RomanAgaltsev/keel/v2/internal/manifest"
)

func TestEmitsParsesFromManifestYAML(t *testing.T) {
	var m manifest.Manifest
	require.NoError(t, yaml.Unmarshal([]byte(`
name: demo
emits:
  checks: [lint]
  actions: [arduino/setup-task]
  needs: [can_approve_pull_request_reviews]
`), &m))

	require.Equal(t, []string{"lint"}, m.Emits.Checks)
	require.Equal(t, []string{"arduino/setup-task"}, m.Emits.Actions)
	require.Equal(t, []string{"can_approve_pull_request_reviews"}, m.Emits.Needs)
}

func TestEmitsAbsentMeansContributesNothing(t *testing.T) {
	var m manifest.Manifest
	require.NoError(t, yaml.Unmarshal([]byte("name: demo\n"), &m))
	require.Empty(t, m.Emits.Checks)
	require.Empty(t, m.Emits.Actions)
	require.Empty(t, m.Emits.Needs)
	require.NoError(t, m.Emits.Validate())
}

func TestEmitsValidateRejectsUnknownNeed(t *testing.T) {
	// The vocabulary is closed on purpose: an unrecognised need would be
	// silently ignored by the settings template, which is the failure mode
	// this whole contract exists to remove.
	e := manifest.Emits{Needs: []string{"can_aprove_pull_request_reviews"}}
	err := e.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "can_aprove_pull_request_reviews")
}

func TestEmitsValidateRejectsMalformedEntries(t *testing.T) {
	cases := []struct {
		name  string
		emits manifest.Emits
		want  string
	}{
		{"empty check", manifest.Emits{Checks: []string{""}}, "empty"},
		{"empty action", manifest.Emits{Actions: []string{""}}, "empty"},
		{"duplicate check", manifest.Emits{Checks: []string{"lint", "lint"}}, "duplicate"},
		{"duplicate action", manifest.Emits{Actions: []string{"a/b", "a/b"}}, "duplicate"},
		{"action carries a version", manifest.Emits{Actions: []string{"a/b@v3"}}, "without a version"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.emits.Validate()
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestEmitsValidateAcceptsAWellFormedBlock(t *testing.T) {
	// The docker:// form is real -- security.yml uses
	// docker://rhysd/actionlint:1.7.7 -- so a Validate that insisted on
	// exactly one slash would reject a module keel ships.
	e := manifest.Emits{
		Checks:  []string{"lint", "test"},
		Actions: []string{"arduino/setup-task", "docker://rhysd/actionlint"},
		Needs:   []string{manifest.NeedCanApprovePullRequestReviews},
	}
	require.NoError(t, e.Validate())
}
