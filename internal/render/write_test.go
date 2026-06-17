package render

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWritePlanAtomic(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "out")
	plan := Plan{Files: map[string]string{
		"README.md":   "hi",
		"pkg/main.go": "package main",
	}}
	require.NoError(t, WritePlan(plan, dir))

	b, err := os.ReadFile(filepath.Join(dir, "README.md"))
	require.NoError(t, err)
	require.Equal(t, "hi", string(b))

	b, err = os.ReadFile(filepath.Join(dir, "pkg", "main.go"))
	require.NoError(t, err)
	require.Equal(t, "package main", string(b))
}

func TestWritePlanRefusesExistingTarget(t *testing.T) {
	dir := t.TempDir() // already exists
	require.Error(t, WritePlan(Plan{Files: map[string]string{"a": "b"}}, dir))
}
