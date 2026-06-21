package outdated

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsePins(t *testing.T) {
	files := map[string][]byte{
		".github/workflows/lint.yml": []byte(
			"steps:\n" +
				"  - uses: actions/checkout@v5\n" +
				"  - uses: github/codeql-action/init@v3\n" +
				"  - uses: docker://rhysd/actionlint:1.7.7\n" + // deferred — ignored
				"  - uses: ./local-action\n" + // local — ignored
				"    with:\n" +
				"      version: v2.3.0 # golangci-lint\n",
		),
		"Taskfile.yml": []byte("vars:\n  GOLANGCI_VERSION: v2.3.0\n"),
	}
	pins := ParsePins(files)

	got := map[string]Pin{}
	for _, p := range pins {
		got[p.Ref] = p
	}
	require.Contains(t, got, "actions/checkout")
	require.True(t, got["actions/checkout"].Major)
	require.Equal(t, "v5", got["actions/checkout"].Current)

	require.Contains(t, got, "github/codeql-action/init")
	require.Equal(t, "github/codeql-action", got["github/codeql-action/init"].Repo())

	require.Contains(t, got, "golangci/golangci-lint")
	require.False(t, got["golangci/golangci-lint"].Major)
	require.Equal(t, "v2.3.0", got["golangci/golangci-lint"].Current)

	require.NotContains(t, got, "docker://rhysd/actionlint")
	require.NotContains(t, got, "./local-action")
}
