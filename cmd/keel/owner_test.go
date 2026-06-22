package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFirstEnv(t *testing.T) {
	t.Setenv("KEEL_A", "")
	t.Setenv("KEEL_B", "second")
	require.Equal(t, "second", firstEnv("KEEL_A", "KEEL_B")) // skips empty, takes first set
	require.Equal(t, "", firstEnv("KEEL_A"))                 // all empty
	require.Equal(t, "", firstEnv())                         // no keys
}

func TestOwnerFromModulePath(t *testing.T) {
	cases := map[string]string{
		"github.com/Acme/repo":     "Acme",
		"github.com/Acme/repo/sub": "Acme", // extra path segments ignored
		"example.com/x/y":          "x",
		"github.com/repo":          "", // too short
		"single":                   "",
		"":                         "",
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			require.Equal(t, want, ownerFromModulePath(in))
		})
	}
}

func TestOwnerOrEnv(t *testing.T) {
	t.Setenv("KEEL_GITHUB_OWNER", "EnvOwner")
	require.Equal(t, "EnvOwner", ownerOrEnv("github.com/Acme/repo")) // env wins

	t.Setenv("KEEL_GITHUB_OWNER", "")
	require.Equal(t, "Acme", ownerOrEnv("github.com/Acme/repo")) // falls back to module path
	require.Equal(t, "", ownerOrEnv("single"))                   // neither available
}
