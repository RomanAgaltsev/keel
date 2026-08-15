package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2/internal/lock"
)

// seedPreHealthLock writes a .scaffold.lock as an older keel wrote it: go-service
// without any of the modules the recipe gained in 2.0.0.
func seedPreHealthLock(t *testing.T, repo string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".scaffold.lock"), []byte(`lock_version: 2
keel_version: 1.7.1
recipe: go-service
modules:
    - name: base-layout
      source: builtin
      version: 1.0.0
    - name: go-mod
      source: builtin
      version: 1.0.0
answers:
    repo_name: demo
    description: a demo service
    module_path: github.com/acme/demo
    author_name: Ada Lovelace
    author_email: ada@example.com
    license: MIT
    provider: github
    visibility: public
    create_remote: false
    dep_bot: dependabot
    enable_codeql: true
    enable_govulncheck: true
    enable_codecov: false
`), 0o600))
}

// runUpdateForTest runs `keel update --path <repo>` and returns its output.
func runUpdateForTest(t *testing.T, repo string, args ...string) (string, error) {
	t.Helper()
	cmd := newUpdateCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(append([]string{"--path", repo}, args...))
	err := cmd.Execute()
	return buf.String(), err
}

// TestUpdateAddsModulesTheRecipeGained is the scenario that motivated 2.1.0: a
// repo scaffolded before the recipe gained the repo-health modules must receive
// them on update. Before this release it received nothing.
func TestUpdateAddsModulesTheRecipeGained(t *testing.T) {
	repo := t.TempDir()
	seedPreHealthLock(t, repo)

	out, err := runUpdateForTest(t, repo)
	require.NoError(t, err, out)

	for _, path := range []string{"LICENSE", "CONTRIBUTING.md", "SECURITY.md", ".editorconfig"} {
		require.FileExists(t, filepath.Join(repo, path), "expected %s from a recipe-gained module", path)
	}
	require.Contains(t, out, "added module")
	require.Contains(t, out, "license")
}

// TestUpdateIsIdempotentAcrossAddition guards §8 of the spec: the added modules
// must be recorded in the lock, or every subsequent update would re-announce
// them forever.
func TestUpdateIsIdempotentAcrossAddition(t *testing.T) {
	repo := t.TempDir()
	seedPreHealthLock(t, repo)

	_, err := runUpdateForTest(t, repo)
	require.NoError(t, err)

	out, err := runUpdateForTest(t, repo)
	require.NoError(t, err, out)
	require.NotContains(t, out, "added module", "the second run has nothing to add")

	lk, err := lock.Read(filepath.Join(repo, ".scaffold.lock"))
	require.NoError(t, err)
	names := make([]string, 0, len(lk.Modules))
	for _, m := range lk.Modules {
		names = append(names, m.Name)
	}
	require.Contains(t, names, "license", "an added module must be recorded")
}
