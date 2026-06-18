package scaffold_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/git"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/provider"
	"github.com/RomanAgaltsev/keel/internal/scaffold"
)

func baseOpts(target string, p provider.Provider) scaffold.Options {
	return scaffold.Options{
		Target:       target,
		Recipe:       "go-service",
		ModuleNames:  []string{"base-layout", "go-mod"},
		Loader:       module.NewFSLoader(keel.BuiltinFS),
		Provider:     p,
		CreateRemote: true,
		Answers: answers.Answers{
			"repo_name": "demo", "description": "d", "module_path": "github.com/x/demo",
			"author_name": "Roman", "author_email": "roman-agalcev@yandex.ru",
		},
	}
}

// State 1: local absent, remote absent → fresh scaffold + create + push to the new
// (empty) remote.
func TestRunFresh(t *testing.T) {
	target := filepath.Join(t.TempDir(), "demo")
	f := &provider.Fake{Repo: provider.RemoteRepo{CloneURL: emptyBare(t)}}
	res, err := scaffold.Run(context.Background(), baseOpts(target, f))
	require.NoError(t, err)
	require.False(t, res.State.LocalPresent)
	require.True(t, f.Created)
	require.True(t, res.Pushed)
	require.FileExists(t, filepath.Join(target, "go.mod"))
	require.FileExists(t, filepath.Join(target, ".scaffold.lock"))
	require.True(t, git.New(target).IsRepo())
}

// State 2: local present, remote absent → overlay (skip-existing) + create + push.
func TestRunOverlayExistingLocal(t *testing.T) {
	target := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(target, "go.mod"), []byte("MINE"), 0o644))
	f := &provider.Fake{Repo: provider.RemoteRepo{CloneURL: emptyBare(t)}}

	res, err := scaffold.Run(context.Background(), baseOpts(target, f))
	require.NoError(t, err)
	require.True(t, res.State.LocalPresent)
	require.Contains(t, res.Skipped, "go.mod") // kept the user's file
	b, _ := os.ReadFile(filepath.Join(target, "go.mod"))
	require.Equal(t, "MINE", string(b))
	require.True(t, f.Created)
}

// Existing EMPTY dir, no remote → overlay path (NOT the fresh atomic write,
// which would refuse the existing dir). Regression test for the empty-dir bug.
func TestRunFreshIntoEmptyDir(t *testing.T) {
	target := t.TempDir() // exists but empty
	opts := baseOpts(target, &provider.Fake{})
	opts.CreateRemote = false
	res, err := scaffold.Run(context.Background(), opts)
	require.NoError(t, err)
	require.False(t, res.State.LocalPresent)
	require.FileExists(t, filepath.Join(target, "go.mod"))
	require.FileExists(t, filepath.Join(target, ".scaffold.lock"))
	require.True(t, git.New(target).IsRepo())
}

// Dry-run touches neither disk nor network.
func TestRunDryRunWritesNothing(t *testing.T) {
	target := filepath.Join(t.TempDir(), "demo")
	opts := baseOpts(target, &provider.Fake{})
	opts.CreateRemote = false
	opts.DryRun = true
	res, err := scaffold.Run(context.Background(), opts)
	require.NoError(t, err)
	require.True(t, res.DryRun)
	require.NotEmpty(t, res.Written)
	require.NoDirExists(t, target) // nothing was created
}

// Re-running over an already-scaffolded tree skips every file and makes no
// empty commit (idempotent).
func TestRunRerunIsIdempotent(t *testing.T) {
	target := filepath.Join(t.TempDir(), "demo")
	opts := baseOpts(target, &provider.Fake{})
	opts.CreateRemote = false
	_, err := scaffold.Run(context.Background(), opts)
	require.NoError(t, err)

	res, err := scaffold.Run(context.Background(), opts) // second run
	require.NoError(t, err)
	require.False(t, res.Committed) // nothing new to commit
}

// State 4: local present, remote present → overlay, set origin, NO push.
func TestRunBothExistNoPush(t *testing.T) {
	target := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(target, "keep.txt"), []byte("x"), 0o644))
	f := &provider.Fake{Exists: true, Repo: provider.RemoteRepo{CloneURL: localBare(t)}}

	res, err := scaffold.Run(context.Background(), baseOpts(target, f))
	require.NoError(t, err)
	require.True(t, res.State.RemotePresent)
	require.False(t, f.Created)        // remote already existed
	require.False(t, res.Pushed)       // never force-pushes
	require.NotEmpty(t, res.NextSteps) // printed reconcile guidance
}

// emptyBare creates an EMPTY bare repo (the shape of a freshly created remote),
// so a first push fast-forwards rather than being rejected.
func emptyBare(t *testing.T) string {
	t.Helper()
	bare := filepath.Join(t.TempDir(), "empty.git")
	out, err := exec.Command("git", "init", "--bare", "-b", "main", bare).CombinedOutput()
	require.NoError(t, err, string(out))
	return bare
}

// localBare creates a bare repo WITH a seed commit (the shape of a pre-existing
// remote), usable as a clone source.
func localBare(t *testing.T) string {
	t.Helper()
	work := t.TempDir()
	r := git.New(work)
	require.NoError(t, r.Init("main"))
	require.NoError(t, r.SetIdentity("Roman Agaltsev", "roman-agalcev@yandex.ru"))
	require.NoError(t, os.WriteFile(filepath.Join(work, "seed.txt"), []byte("s"), 0o644))
	require.NoError(t, r.AddAll())
	require.NoError(t, r.Commit("seed"))
	bare := filepath.Join(t.TempDir(), "origin.git")
	_, err := r.Run("clone", "--bare", work, bare)
	require.NoError(t, err)
	return bare
}
