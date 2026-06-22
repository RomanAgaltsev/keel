package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/lock"
)

func TestOutdatedModulesOnly(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, lock.Write(filepath.Join(dir, ".scaffold.lock"), lock.Lock{
		Recipe:  "go-service",
		Modules: []lock.Module{{Name: "lint-go", Source: "builtin", Version: "0.9.0"}},
	}))

	cmd := newOutdatedCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--path", dir, "--modules-only"})
	err := cmd.Execute()

	require.ErrorContains(t, err, "updates available") // non-zero exit signal
	s := out.String()
	require.Contains(t, s, "lint-go")
	require.Contains(t, s, "0.9.0")
}

func TestOutdatedModulesOnlyClean(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, lock.Write(filepath.Join(dir, ".scaffold.lock"), lock.Lock{
		Modules: []lock.Module{{Name: "ext", Source: "git", Version: "0.1.0"}}, // non-builtin → skipped
	}))
	cmd := newOutdatedCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--path", dir, "--modules-only"})
	require.NoError(t, cmd.Execute()) // nothing outdated → no error
}

func TestReadPinFiles(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(wfDir, 0o750))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte("version: '3'\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(wfDir, "ci.yml"), []byte("name: ci\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(wfDir, "release.yaml"), []byte("name: release\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(wfDir, "notes.txt"), []byte("ignored\n"), 0o644)) // non-yaml skipped
	require.NoError(t, os.MkdirAll(filepath.Join(wfDir, "sub"), 0o750))                             // dir skipped

	files, err := readPinFiles(dir)
	require.NoError(t, err)

	require.Contains(t, files, "Taskfile.yml")
	require.Contains(t, files, ".github/workflows/ci.yml")
	require.Contains(t, files, ".github/workflows/release.yaml")
	require.NotContains(t, files, ".github/workflows/notes.txt")
	require.Len(t, files, 3)
}

func TestReadPinFilesNoWorkflows(t *testing.T) {
	dir := t.TempDir() // no Taskfile, no .github/workflows
	files, err := readPinFiles(dir)
	require.NoError(t, err)
	require.Empty(t, files)
}
