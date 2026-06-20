package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBumpModuleRewritesVersionLine(t *testing.T) {
	dir := t.TempDir()
	md := filepath.Join(dir, "modules", "lint")
	require.NoError(t, os.MkdirAll(md, 0o755))
	const src = "name: lint\ndescription: x\nversion: 1.0.0\nlanguage: go\n"
	require.NoError(t, os.WriteFile(filepath.Join(md, "module.yaml"), []byte(src), 0o644))

	t.Chdir(dir)
	got, err := bumpModule("lint", "patch")
	require.NoError(t, err)
	require.Equal(t, "1.0.1", got)

	b, _ := os.ReadFile(filepath.Join(md, "module.yaml"))
	require.Contains(t, string(b), "version: 1.0.1")
	require.Contains(t, string(b), "description: x") // other lines preserved
}
