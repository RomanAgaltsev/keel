package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/lock"
)

func TestOutdatedModulesOnly(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, lock.Write(filepath.Join(dir, ".scaffold.lock"), lock.Lock{
		Recipe:  "go-service",
		Modules: []lock.Module{{Name: "lint", Source: "builtin", Version: "0.9.0"}},
	}))

	cmd := newOutdatedCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--path", dir, "--modules-only"})
	err := cmd.Execute()

	require.ErrorContains(t, err, "updates available") // non-zero exit signal
	s := out.String()
	require.Contains(t, s, "lint")
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
