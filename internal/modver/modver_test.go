package modver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModulesTouched(t *testing.T) {
	got := ModulesTouched([]string{
		"modules/lint/templates/.github/workflows/lint.yml",
		"modules/lint/module.yaml",
		"modules/security/templates/x.yml",
		"internal/render/render.go", // ignored
		"README.md",                 // ignored
	})
	require.Equal(t, []string{"lint", "security"}, got)
}

func TestOffenders(t *testing.T) {
	touched := []string{"lint", "security", "newmod"}
	old := map[string]string{"lint": "1.0.0", "security": "1.2.0"} // newmod absent => new
	new := map[string]string{"lint": "1.0.0", "security": "1.2.1", "newmod": "1.0.0"}
	// lint unchanged => offender; security bumped => ok; newmod new => ok.
	require.Equal(t, []string{"lint"}, Offenders(touched, old, new))
}

func TestBump(t *testing.T) {
	cases := []struct{ in, level, want string }{
		{"1.0.0", "patch", "1.0.1"},
		{"1.2.3", "minor", "1.3.0"},
		{"1.2.3", "major", "2.0.0"},
		{"v1.0.0", "patch", "v1.0.1"},
	}
	for _, c := range cases {
		got, err := Bump(c.in, c.level)
		require.NoError(t, err)
		require.Equal(t, c.want, got)
	}
	_, err := Bump("1.0", "patch")
	require.Error(t, err)
	_, err = Bump("1.0.0", "nope")
	require.Error(t, err)
}

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.1", -1},
		{"1.2.0", "1.1.9", 1},
		{"v2.3.0", "v2.3.0", 0},
		{"v2.3.0", "2.3.1", -1}, // mixed v-prefix
	}
	for _, c := range cases {
		got, err := Compare(c.a, c.b)
		require.NoError(t, err)
		require.Equal(t, c.want, got, "%s vs %s", c.a, c.b)
	}
	_, err := Compare("1.0", "1.0.0")
	require.Error(t, err)
}
