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
