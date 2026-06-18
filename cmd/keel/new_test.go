package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// seedBare makes a local bare repo (clone source) and returns its path.
func seedBare(t *testing.T) string {
	t.Helper()
	work := t.TempDir()
	run := func(args ...string) {
		out, err := exec.Command("git", append([]string{"-C", work}, args...)...).CombinedOutput()
		require.NoError(t, err, string(out))
	}
	run("init", "-b", "main")
	run("config", "user.name", "Roman Agaltsev")
	run("config", "user.email", "roman-agalcev@yandex.ru")
	require.NoError(t, os.WriteFile(filepath.Join(work, "seed.txt"), []byte("s"), 0o644))
	run("add", "-A")
	run("commit", "-m", "seed")
	bare := filepath.Join(t.TempDir(), "origin.git")
	out, err := exec.Command("git", "-C", work, "clone", "--bare", work, bare).CombinedOutput()
	require.NoError(t, err, string(out))
	return bare
}

// State 3: local absent + remote present → clone-then-overlay.
func TestNewCloneThenOverlay(t *testing.T) {
	bare := seedBare(t)
	target := filepath.Join(t.TempDir(), "demo")

	cmd := newNewCmd()
	cmd.SetArgs([]string{
		"--no-input",
		"--answers", writeAnswers(t),
		"--remote-url", bare,
		"--target", target,
	})
	require.NoError(t, cmd.Execute())

	// Came from the clone (seed.txt) AND got the overlay (go.mod, skip-existing).
	require.FileExists(t, filepath.Join(target, "seed.txt"))
	require.FileExists(t, filepath.Join(target, "go.mod"))
	require.FileExists(t, filepath.Join(target, ".scaffold.lock"))
}

func writeAnswers(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "answers.yaml")
	require.NoError(t, os.WriteFile(p, []byte(`repo_name: demo
description: d
module_path: github.com/x/demo
author_name: Roman Agaltsev
author_email: roman-agalcev@yandex.ru
license: MIT
visibility: public
provider: none
create_remote: false
`), 0o644))
	return p
}
