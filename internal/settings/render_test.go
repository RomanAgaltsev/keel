package settings_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2/internal/settings"
)

func TestRenderInSync(t *testing.T) {
	var buf bytes.Buffer
	settings.Report{Applied: true}.Render(&buf)
	require.Contains(t, buf.String(), "settings: in sync")
}

func TestRenderApplied(t *testing.T) {
	var buf bytes.Buffer
	settings.Report{
		Applied: true,
		Changes: []settings.Change{{Key: "repository.has_wiki", From: "true", To: "false"}},
	}.Render(&buf)

	out := buf.String()
	require.Contains(t, out, "applied 1 setting")
	require.Contains(t, out, "repository.has_wiki: true -> false")
}

func TestRenderCheckModeSaysWouldChange(t *testing.T) {
	var buf bytes.Buffer
	settings.Report{
		Changes: []settings.Change{{Key: "repository.has_wiki", From: "true", To: "false"}},
	}.Render(&buf)
	require.Contains(t, buf.String(), "drift: 1 setting")
}

func TestRenderFailuresAndUnsupported(t *testing.T) {
	var buf bytes.Buffer
	settings.Report{
		Applied:     true,
		Failed:      []settings.Failure{{Group: "ruleset", Reason: "token lacks administration:write"}},
		Unsupported: []settings.Unsupported{{Key: "security.dependency_graph", Provider: "gitlab", Reason: "no equivalent"}},
	}.Render(&buf)

	out := buf.String()
	require.Contains(t, out, "! ruleset: token lacks administration:write")
	require.Contains(t, out, "- security.dependency_graph: not supported on gitlab (no equivalent)")
}
