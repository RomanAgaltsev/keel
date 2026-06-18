package git_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/git"
)

func TestInitCommitAndIsRepo(t *testing.T) {
	dir := t.TempDir()
	r := git.New(dir)
	require.False(t, r.IsRepo())

	require.NoError(t, r.Init("main"))
	require.True(t, r.IsRepo())
	require.NoError(t, r.SetIdentity("Roman Agaltsev", "roman-agalcev@yandex.ru"))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644))
	require.NoError(t, r.AddAll())
	require.NoError(t, r.Commit("initial commit"))

	out, err := r.Run("log", "--oneline")
	require.NoError(t, err)
	require.Contains(t, out, "initial commit")

	// After committing, a clean tree reports no changes.
	clean, err := r.HasChanges()
	require.NoError(t, err)
	require.False(t, clean)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.txt"), []byte("z"), 0o644))
	dirty, err := r.HasChanges()
	require.NoError(t, err)
	require.True(t, dirty)
}

func TestCloneFromLocalBare(t *testing.T) {
	// Build a source bare repo with one commit.
	work := t.TempDir()
	src := git.New(work)
	require.NoError(t, src.Init("main"))
	require.NoError(t, src.SetIdentity("Roman Agaltsev", "roman-agalcev@yandex.ru"))
	require.NoError(t, os.WriteFile(filepath.Join(work, "f.txt"), []byte("hi"), 0o644))
	require.NoError(t, src.AddAll())
	require.NoError(t, src.Commit("seed"))

	bare := filepath.Join(t.TempDir(), "origin.git")
	_, err := git.New(work).Run("clone", "--bare", work, bare)
	require.NoError(t, err)

	dst := filepath.Join(t.TempDir(), "checkout")
	r, err := git.Clone(bare, dst)
	require.NoError(t, err)
	require.True(t, r.IsRepo())
	_, err = os.Stat(filepath.Join(dst, "f.txt"))
	require.NoError(t, err)
}
