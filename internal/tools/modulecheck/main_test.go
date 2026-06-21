package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

func writeModule(t *testing.T, dir, name, version string) {
	t.Helper()
	md := filepath.Join(dir, "modules", name)
	require.NoError(t, os.MkdirAll(md, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(md, "module.yaml"),
		[]byte("name: "+name+"\nversion: "+version+"\n"), 0o644))
}

// run() resolves paths relative to CWD, so each case chdirs into the temp repo.
func TestRunDetectsMissingBump(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-b", "main")
	git(t, dir, "config", "user.email", "t@t")
	git(t, dir, "config", "user.name", "t")
	writeModule(t, dir, "lint-go", "1.0.0")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "base")
	git(t, dir, "checkout", "-b", "feature")

	// Change a template file but NOT the version.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "modules", "lint-go", "x.txt"), []byte("y"), 0o644))
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "change without bump")

	t.Chdir(dir)
	require.Error(t, run("main"), "should fail: lint changed without a bump")

	// Now bump the version → passes.
	writeModule(t, dir, "lint-go", "1.0.1")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "bump")
	require.NoError(t, run("main"))
}
