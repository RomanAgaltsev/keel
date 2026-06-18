package scaffold

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalPresent(t *testing.T) {
	// Missing dir → absent.
	missing := filepath.Join(t.TempDir(), "nope")
	present, err := localPresent(missing)
	require.NoError(t, err)
	require.False(t, present)

	// Empty dir → absent.
	empty := t.TempDir()
	present, err = localPresent(empty)
	require.NoError(t, err)
	require.False(t, present)

	// Non-empty dir → present.
	require.NoError(t, os.WriteFile(filepath.Join(empty, "x"), []byte("y"), 0o644))
	present, err = localPresent(empty)
	require.NoError(t, err)
	require.True(t, present)
}

func TestPathExists(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	ok, err := pathExists(missing)
	require.NoError(t, err)
	require.False(t, ok)

	empty := t.TempDir() // exists but empty: localPresent=false, pathExists=true
	ok, err = pathExists(empty)
	require.NoError(t, err)
	require.True(t, ok)
}
