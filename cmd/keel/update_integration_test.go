package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/RomanAgaltsev/keel/internal/lock"
)

// TestUpdateRoundTrip scaffolds a real repo, forces the lock to look "behind" by
// rewriting recorded module versions to 0.0.1, edits one tracked file, and runs
// keel update — then asserts the hash-aware outcomes and idempotency.
func TestUpdateRoundTrip(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "svc")

	// 1. Scaffold a real go-service non-interactively.
	answersFile := writeAnswersFile(t) // helper: dump fullAnswers() to a temp YAML
	root := newRootCmd()
	root.SetArgs([]string{
		"new", "--recipe", "go-service", "--target", repo,
		"--answers", answersFile, "--no-input",
	})
	require.NoError(t, root.Execute())

	// 2. Make every module look behind, so all become update candidates.
	lockPath := filepath.Join(repo, ".scaffold.lock")
	lk, err := lock.Read(lockPath)
	require.NoError(t, err)
	for i := range lk.Modules {
		lk.Modules[i].Version = "0.0.1"
	}
	require.NoError(t, lock.Write(lockPath, lk))

	// 3. Edit one tracked file (pick the first recorded file of the first module).
	edited := lk.Modules[0].Files[0].Path
	require.NoError(t, os.WriteFile(filepath.Join(repo, filepath.FromSlash(edited)),
		[]byte("# locally edited\n"), 0o644))

	// 4. Run update.
	root = newRootCmd()
	root.SetArgs([]string{"update", "--path", repo})
	require.NoError(t, root.Execute())

	// Edited file preserved; its new render is beside it.
	b, _ := os.ReadFile(filepath.Join(repo, filepath.FromSlash(edited)))
	require.Equal(t, "# locally edited\n", string(b))
	_, statErr := os.Stat(filepath.Join(repo, filepath.FromSlash(edited)+".keel-new"))
	require.NoError(t, statErr)

	// Lock bumped off 0.0.1 for at least the first module.
	after, err := lock.Read(lockPath)
	require.NoError(t, err)
	require.NotEqual(t, "0.0.1", after.Modules[0].Version)

	// 5. Idempotent: a second run on the now-current lock reports nothing to do
	//    (the edited file remains a conflict, but versions are no longer behind).
	root = newRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"update", "--path", repo, "--dry-run"})
	require.NoError(t, root.Execute())
	require.Contains(t, buf.String(), "up to date")
}

// writeAnswersFile marshals fullAnswers() to a temp YAML file and returns its path.
func writeAnswersFile(t *testing.T) string {
	t.Helper()
	b, err := yaml.Marshal(map[string]any(fullAnswers()))
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "answers.yaml")
	require.NoError(t, os.WriteFile(path, b, 0o644))
	return path
}
