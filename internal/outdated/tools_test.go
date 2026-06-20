package outdated

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeRC map[string]string // repo -> latest tag; "" entry → ErrNoRelease

func (f fakeRC) LatestTag(_ context.Context, repo string) (string, error) {
	tag, ok := f[repo]
	if !ok || tag == "" {
		return "", ErrNoRelease
	}
	return tag, nil
}

func TestToolUpdates(t *testing.T) {
	pins := []Pin{
		{Ref: "actions/checkout", Current: "v5", Major: true}, // latest v6 → outdated
		{Ref: "actions/setup-go", Current: "v6", Major: true}, // latest v6 → current
		{Ref: "golangci/golangci-lint", Current: "v2.3.0"},    // latest v2.4.0 → outdated
		{Ref: "some/norelease", Current: "v1", Major: true},   // no release → skipped
	}
	rc := fakeRC{
		"actions/checkout":       "v6.1.0",
		"actions/setup-go":       "v6.0.2",
		"golangci/golangci-lint": "v2.4.0",
	}
	ups, skipped := ToolUpdates(context.Background(), pins, rc)
	require.Equal(t, 1, skipped)

	got := map[string]ToolUpdate{}
	for _, u := range ups {
		got[u.Repo] = u
	}
	require.Contains(t, got, "actions/checkout")
	require.Equal(t, "v6.1.0", got["actions/checkout"].Latest)
	require.Contains(t, got, "golangci/golangci-lint")
	require.NotContains(t, got, "actions/setup-go") // not behind
}

func TestReportEmpty(t *testing.T) {
	require.True(t, Report{}.Empty())
	require.False(t, Report{Tools: []ToolUpdate{{Repo: "x"}}}.Empty())
	require.False(t, Report{Modules: []ModuleUpdate{{Name: "y"}}}.Empty())
}
