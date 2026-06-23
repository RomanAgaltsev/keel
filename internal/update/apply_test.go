package update_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/update"
)

func TestApplyWritesByClass(t *testing.T) {
	target := t.TempDir()
	// Pre-existing user-edited file for the Conflict case.
	require.NoError(t, os.WriteFile(filepath.Join(target, "edited.yml"), []byte("mine"), 0o644))

	p := update.Plan{Changes: []update.FileChange{
		{Path: "clean.yml", Class: update.Clean, Content: "fresh"},
		{Path: "sub/new.yml", Class: update.New, Content: "added"},
		{Path: "edited.yml", Class: update.Conflict, Content: "theirs"},
		{Path: "gone.yml", Class: update.Removed},
	}}

	got, err := update.Apply(p, target, false)
	require.NoError(t, err)
	require.Equal(t, []string{"clean.yml"}, got.Updated)
	require.Equal(t, []string{"sub/new.yml"}, got.New)
	require.Equal(t, []string{"edited.yml"}, got.Conflicts)
	require.Equal(t, []string{"gone.yml"}, got.Removed)

	// Clean + New written verbatim.
	requireFile(t, filepath.Join(target, "clean.yml"), "fresh")
	requireFile(t, filepath.Join(target, "sub", "new.yml"), "added")
	// Conflict: user's file untouched, new render beside it.
	requireFile(t, filepath.Join(target, "edited.yml"), "mine")
	requireFile(t, filepath.Join(target, "edited.yml.keel-new"), "theirs")
}

func TestApplyOverwriteReplacesConflict(t *testing.T) {
	target := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(target, "edited.yml"), []byte("mine"), 0o644))

	p := update.Plan{Changes: []update.FileChange{
		{Path: "edited.yml", Class: update.Conflict, Content: "theirs"},
	}}
	got, err := update.Apply(p, target, true) // overwrite
	require.NoError(t, err)
	require.Equal(t, []string{"edited.yml"}, got.Updated) // counted as updated, not conflict
	require.Empty(t, got.Conflicts)
	requireFile(t, filepath.Join(target, "edited.yml"), "theirs")
	_, statErr := os.Stat(filepath.Join(target, "edited.yml.keel-new"))
	require.True(t, os.IsNotExist(statErr)) // no sibling under overwrite
}

func requireFile(t *testing.T, path, want string) {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, want, string(b))
}
