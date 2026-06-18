package lock_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/lock"
)

func TestWriteThenRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".scaffold.lock")
	l := lock.Lock{
		KeelVersion: "0.1.0",
		Recipe:      "go-service",
		Modules: []lock.Module{
			{Name: "base-layout", Source: "builtin", Version: "1.0.0"},
		},
		Answers: map[string]any{"repo_name": "foo"},
	}
	require.NoError(t, lock.Write(path, l))

	got, err := lock.Read(path)
	require.NoError(t, err)
	require.Equal(t, l.Recipe, got.Recipe)
	require.Equal(t, "base-layout", got.Modules[0].Name)
	require.Equal(t, "foo", got.Answers["repo_name"])
}
