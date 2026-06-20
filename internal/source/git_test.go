package source_test

import (
	"context"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/recipe"
	"github.com/RomanAgaltsev/keel/internal/source"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

// bareRepoWithModule builds a bare repo containing modules/logging/{module.yaml,
// templates/} at <subdir> and returns its path plus the tag "v1".
func bareRepoWithModule(t *testing.T) (url, ref string) {
	t.Helper()
	work := t.TempDir()
	git(t, work, "init", "-b", "main")
	git(t, work, "config", "user.email", "t@t")
	git(t, work, "config", "user.name", "t")
	md := filepath.Join(work, "logging")
	require.NoError(t, os.MkdirAll(filepath.Join(md, "templates"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(md, "module.yaml"),
		[]byte("name: logging\nversion: 1.2.0\nlanguage: go\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(md, "templates", "log.go.tmpl"),
		[]byte("package log\n"), 0o644))
	git(t, work, "add", "-A")
	git(t, work, "commit", "-m", "module")
	git(t, work, "tag", "v1")

	bare := filepath.Join(t.TempDir(), "origin.git")
	out, err := exec.Command("git", "clone", "--bare", work, bare).CombinedOutput()
	require.NoError(t, err, string(out))
	return bare, "v1"
}

func TestResolveGit(t *testing.T) {
	url, ref := bareRepoWithModule(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir()) // isolate the module cache (Linux/macOS)

	res, err := source.Resolve(context.Background(),
		recipe.Source{Git: url, Subdir: "logging", Ref: ref}, "")
	require.NoError(t, err)
	require.Contains(t, res.Source, "git:")
	require.Contains(t, res.Source, "//logging@v1")
	require.Len(t, res.Version, 40) // resolved commit SHA

	b, err := fs.ReadFile(res.FS, "module.yaml")
	require.NoError(t, err)
	require.Contains(t, string(b), "name: logging")
}

func TestResolveGitMissingSubdir(t *testing.T) {
	url, ref := bareRepoWithModule(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	_, err := source.Resolve(context.Background(),
		recipe.Source{Git: url, Subdir: "nope", Ref: ref}, "")
	require.Error(t, err)
}

// TestResolveGitRefreshesBranch is the regression test for the stale-cache bug:
// a branch ref must follow the upstream tip across resolves, not pin to the first
// clone.
func TestResolveGitRefreshesBranch(t *testing.T) {
	work := t.TempDir()
	git(t, work, "init", "-b", "main")
	git(t, work, "config", "user.email", "t@t")
	git(t, work, "config", "user.name", "t")
	md := filepath.Join(work, "logging")
	require.NoError(t, os.MkdirAll(filepath.Join(md, "templates"), 0o755))
	writeMod := func(ver string) {
		require.NoError(t, os.WriteFile(filepath.Join(md, "module.yaml"),
			[]byte("name: logging\nversion: "+ver+"\nlanguage: go\n"), 0o644))
	}
	writeMod("1.2.0")
	require.NoError(t, os.WriteFile(filepath.Join(md, "templates", "log.go.tmpl"),
		[]byte("package log\n"), 0o644))
	git(t, work, "add", "-A")
	git(t, work, "commit", "-m", "v1")

	bare := filepath.Join(t.TempDir(), "origin.git")
	out, err := exec.Command("git", "clone", "--bare", work, bare).CombinedOutput()
	require.NoError(t, err, string(out))

	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	src := recipe.Source{Git: bare, Subdir: "logging", Ref: "main"}

	res1, err := source.Resolve(context.Background(), src, "")
	require.NoError(t, err)

	// Advance the branch upstream, then push it to the bare origin.
	writeMod("1.3.0")
	git(t, work, "add", "-A")
	git(t, work, "commit", "-m", "v2")
	git(t, work, "push", bare, "main")

	res2, err := source.Resolve(context.Background(), src, "")
	require.NoError(t, err)
	require.NotEqual(t, res1.Version, res2.Version, "branch ref must refresh to the new SHA")

	b, err := fs.ReadFile(res2.FS, "module.yaml")
	require.NoError(t, err)
	require.Contains(t, string(b), "version: 1.3.0")
}
