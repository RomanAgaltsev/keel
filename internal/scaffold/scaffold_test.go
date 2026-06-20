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
	"github.com/RomanAgaltsev/keel/internal/lock"
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

// Dry-run must not probe the remote (no network): the provider's RepoExists is
// never called, and nothing lands on disk. Regression test for the dry-run network bug.
func TestRunDryRunSkipsRemoteCheck(t *testing.T) {
	target := filepath.Join(t.TempDir(), "demo")
	f := &provider.Fake{} // baseOpts sets CreateRemote: true, which used to trigger a check
	opts := baseOpts(target, f)
	opts.DryRun = true

	res, err := scaffold.Run(context.Background(), opts)
	require.NoError(t, err)
	require.True(t, res.DryRun)
	require.False(t, f.ExistsCalled, "dry-run must not call the provider (no network)")
	require.NoDirExists(t, target)
}

// State 3 via --remote-url (no provider): clone-then-overlay must still push the
// overlay commit. Regression test for the skipped-push bug.
func TestRunRemoteURLClonePushes(t *testing.T) {
	target := filepath.Join(t.TempDir(), "demo")
	opts := baseOpts(target, nil) // no provider; remote supplied by URL
	opts.RemoteURL = localBare(t)

	res, err := scaffold.Run(context.Background(), opts)
	require.NoError(t, err)
	require.True(t, res.State.RemotePresent)
	require.False(t, res.State.LocalPresent)
	require.False(t, res.Created)                            // no provider ⇒ nothing created
	require.True(t, res.Pushed)                              // clone-then-overlay now pushes via the remote URL
	require.FileExists(t, filepath.Join(target, "seed.txt")) // cloned from the remote
	require.FileExists(t, filepath.Join(target, "go.mod"))   // overlaid by keel
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
	ctx := context.Background()
	work := t.TempDir()
	r := git.New(work)
	require.NoError(t, r.Init(ctx, "main"))
	require.NoError(t, r.SetIdentity(ctx, "Roman Agaltsev", "roman-agalcev@yandex.ru"))
	require.NoError(t, os.WriteFile(filepath.Join(work, "seed.txt"), []byte("s"), 0o644))
	require.NoError(t, r.AddAll(ctx))
	require.NoError(t, r.Commit(ctx, "seed"))
	bare := filepath.Join(t.TempDir(), "origin.git")
	_, err := r.Run(ctx, "clone", "--bare", work, bare)
	require.NoError(t, err)
	return bare
}

func TestRunRecordsExternalProvenance(t *testing.T) {
	// External module on local disk.
	ext := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(ext, "templates"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(ext, "module.yaml"),
		[]byte("name: logging\nversion: 1.2.0\nlanguage: go\nfiles:\n  - src: \"*\"\n    dest: \".\"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(ext, "templates", "log.go.tmpl"),
		[]byte("package log\n"), 0o644))

	comp, err := module.NewComposite(keel.BuiltinFS, []module.External{
		{Name: "logging", FS: os.DirFS(ext), Source: "dir:./logging", Version: "1.2.0"},
	})
	require.NoError(t, err)

	target := filepath.Join(t.TempDir(), "demo")
	opts := baseOpts(target, nil)
	opts.CreateRemote = false
	opts.Loader = comp
	opts.ModuleNames = []string{"base-layout", "go-mod", "logging"}

	_, err = scaffold.Run(context.Background(), opts)
	require.NoError(t, err)

	require.FileExists(t, filepath.Join(target, "log.go"))
	lk, err := lock.Read(filepath.Join(target, ".scaffold.lock"))
	require.NoError(t, err)
	got := map[string]string{} // name -> source
	for _, m := range lk.Modules {
		got[m.Name] = m.Source
	}
	require.Equal(t, "dir:./logging", got["logging"])
	require.Equal(t, "builtin", got["base-layout"])
}
