package lock_test

import (
	"os"
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

func TestLockV2RoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".scaffold.lock")
	in := lock.Lock{
		KeelVersion: "1.6.0",
		Recipe:      "go-service",
		Modules: []lock.Module{{
			Name: "lint", Source: "builtin", Version: "1.1.0",
			Files: []lock.File{{Path: ".golangci.yml", SHA256: "ab12"}},
		}},
		Answers: map[string]any{"repo_name": "demo"},
	}
	require.NoError(t, lock.Write(p, in))

	got, err := lock.Read(p)
	require.NoError(t, err)
	require.Equal(t, 2, got.LockVersion) // Write stamps the current schema version
	require.Equal(t, in.Modules, got.Modules)
}

func TestLockReadV1HasNoFiles(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".scaffold.lock")
	// A v1 lock: no lock_version, no files.
	require.NoError(t, os.WriteFile(p, []byte(
		"keel_version: 1.5.0\nrecipe: go-service\nmodules:\n  - name: lint\n    source: builtin\n    version: 1.0.0\nanswers: {}\n"), 0o600))

	got, err := lock.Read(p)
	require.NoError(t, err)
	require.Equal(t, 0, got.LockVersion) // absent ⇒ v1
	require.Nil(t, got.Modules[0].Files)
}

func TestHashBytesStable(t *testing.T) {
	require.Equal(t, lock.HashBytes([]byte("hello")), lock.HashBytes([]byte("hello")))
	require.NotEqual(t, lock.HashBytes([]byte("a")), lock.HashBytes([]byte("b")))
	require.Len(t, lock.HashBytes([]byte("x")), 64) // hex sha256
}
