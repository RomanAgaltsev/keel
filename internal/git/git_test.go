package git_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/git"
)

func TestInitCommitAndIsRepo(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	r := git.New(dir)
	require.False(t, r.IsRepo())

	require.NoError(t, r.Init(ctx, "main"))
	require.True(t, r.IsRepo())
	require.NoError(t, r.SetIdentity(ctx, "Roman Agaltsev", "roman-agalcev@yandex.ru"))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644))
	require.NoError(t, r.AddAll(ctx))
	require.NoError(t, r.Commit(ctx, "initial commit"))

	out, err := r.Run(ctx, "log", "--oneline")
	require.NoError(t, err)
	require.Contains(t, out, "initial commit")

	// After committing, a clean tree reports no changes.
	clean, err := r.HasChanges(ctx)
	require.NoError(t, err)
	require.False(t, clean)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.txt"), []byte("z"), 0o644))
	dirty, err := r.HasChanges(ctx)
	require.NoError(t, err)
	require.True(t, dirty)
}

func TestAddRemoteAndPush(t *testing.T) {
	ctx := context.Background()

	// A local bare repo stands in for the remote — no network needed.
	bare := filepath.Join(t.TempDir(), "origin.git")
	require.NoError(t, os.MkdirAll(bare, 0o750))
	_, err := git.New(bare).Run(ctx, "init", "--bare", "-b", "main")
	require.NoError(t, err)

	dir := t.TempDir()
	r := git.New(dir)
	require.Equal(t, dir, r.Dir()) // Dir reports the working-tree path

	require.NoError(t, r.Init(ctx, "main"))
	require.NoError(t, r.SetIdentity(ctx, "Roman Agaltsev", "roman-agalcev@yandex.ru"))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "f.txt"), []byte("hi"), 0o644))
	require.NoError(t, r.AddAll(ctx))
	require.NoError(t, r.Commit(ctx, "initial commit"))

	require.NoError(t, r.AddRemote(ctx, "origin", bare))
	require.NoError(t, r.AddRemote(ctx, "origin", bare)) // re-add exercises the set-url fallback
	require.NoError(t, r.Push(ctx, "origin", "main"))

	// The push reached the remote: it now advertises refs/heads/main.
	out, err := r.Run(ctx, "ls-remote", "origin")
	require.NoError(t, err)
	require.Contains(t, out, "refs/heads/main")
}

func TestCloneFromLocalBare(t *testing.T) {
	ctx := context.Background()
	// Build a source bare repo with one commit.
	work := t.TempDir()
	src := git.New(work)
	require.NoError(t, src.Init(ctx, "main"))
	require.NoError(t, src.SetIdentity(ctx, "Roman Agaltsev", "roman-agalcev@yandex.ru"))
	require.NoError(t, os.WriteFile(filepath.Join(work, "f.txt"), []byte("hi"), 0o644))
	require.NoError(t, src.AddAll(ctx))
	require.NoError(t, src.Commit(ctx, "seed"))

	bare := filepath.Join(t.TempDir(), "origin.git")
	_, err := git.New(work).Run(ctx, "clone", "--bare", work, bare)
	require.NoError(t, err)

	dst := filepath.Join(t.TempDir(), "checkout")
	r, err := git.Clone(ctx, bare, dst)
	require.NoError(t, err)
	require.True(t, r.IsRepo())
	_, err = os.Stat(filepath.Join(dst, "f.txt"))
	require.NoError(t, err)
}
