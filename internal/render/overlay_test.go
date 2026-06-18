package render

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOverlaySkipsExisting(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("MINE"), 0o644))

	plan := Plan{Files: map[string]string{"README.md": "GENERATED", "Taskfile.yml": "tasks"}}
	res, err := OverlayPlan(plan, dir, false)
	require.NoError(t, err)

	require.Equal(t, []string{"Taskfile.yml"}, res.Written)
	require.Equal(t, []string{"README.md"}, res.Skipped)

	b, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	require.Equal(t, "MINE", string(b)) // untouched
	b, _ = os.ReadFile(filepath.Join(dir, "Taskfile.yml"))
	require.Equal(t, "tasks", string(b))
}

func TestOverlayOverwrite(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("MINE"), 0o644))

	res, err := OverlayPlan(Plan{Files: map[string]string{"README.md": "GENERATED"}}, dir, true)
	require.NoError(t, err)
	require.Equal(t, []string{"README.md"}, res.Written)
	require.Empty(t, res.Skipped)

	b, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	require.Equal(t, "GENERATED", string(b))
}
