package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2/internal/provider"
	"github.com/RomanAgaltsev/keel/v2/internal/settings"
)

// stubGroup returns canned plan results, so the command's exit-code logic can be
// exercised without a provider or a network.
type stubGroup struct {
	changes  []settings.Change
	planErr  error
	applyErr error
}

func (s *stubGroup) Name() string { return "stub" }

func (s *stubGroup) Plan(_ context.Context, _ settings.Desired) ([]settings.Change, error) {
	return s.changes, s.planErr
}

func (s *stubGroup) Apply(_ context.Context) error { return s.applyErr }

// settingsRepo builds a directory with a settings file and a lock naming the
// remote, then points resolveProvider at a fake carrying the given groups.
func settingsRepo(t *testing.T, groups ...settings.Group) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".github"), 0o750))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, filepath.FromSlash(settings.DefaultPath)),
		[]byte("version: 1\nrepository:\n  has_wiki: false\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".scaffold.lock"), []byte(
		"lock_version: 2\nkeel_version: 2.2.0\nrecipe: go-service\nmodules: []\nanswers:\n"+
			"    repo_name: demo\n    module_path: github.com/acme/demo\n    provider: github\n"), 0o600))

	f := &provider.FakeApplier{Groups: groups}
	orig := resolveProvider
	resolveProvider = func(_, _ string) (provider.Provider, error) { return f, nil }
	t.Cleanup(func() { resolveProvider = orig })
	return dir
}

// runSettings drives the real cobra command and returns its output and exit code.
func runSettings(t *testing.T, dir string, args ...string) (string, int) {
	t.Helper()
	cmd := newSettingsCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(append([]string{"apply", "-C", dir}, args...))

	err := cmd.Execute()
	if err == nil {
		return buf.String(), 0
	}
	var ec exitCodeError
	if errors.As(err, &ec) {
		return buf.String(), ec.code
	}
	return buf.String(), 1
}

func TestSettingsCheckExitsOneOnDrift(t *testing.T) {
	dir := settingsRepo(t, &stubGroup{changes: []settings.Change{
		{Key: "repository.has_wiki", From: "true", To: "false"},
	}})

	out, code := runSettings(t, dir, "--check")
	require.Equal(t, 1, code, "drift exits 1")
	require.Contains(t, out, "drift: 1 setting")
}

func TestSettingsCheckExitsZeroWhenInSync(t *testing.T) {
	dir := settingsRepo(t, &stubGroup{})

	out, code := runSettings(t, dir, "--check")
	require.Equal(t, 0, code)
	require.Contains(t, out, "in sync")
}

// TestSettingsCheckExitsTwoOnError separates "could not read" from "differs":
// a --check that cannot see the remote must not be mistaken for clean drift.
func TestSettingsCheckExitsTwoOnError(t *testing.T) {
	dir := settingsRepo(t, &stubGroup{planErr: errors.New("403 forbidden")})

	out, code := runSettings(t, dir, "--check")
	require.Equal(t, 2, code)
	require.Contains(t, out, "! stub: 403 forbidden")
}

func TestSettingsApplyExitsOneOnFailure(t *testing.T) {
	dir := settingsRepo(t, &stubGroup{
		changes:  []settings.Change{{Key: "repository.has_wiki", From: "true", To: "false"}},
		applyErr: errors.New("422 unprocessable"),
	})

	out, code := runSettings(t, dir)
	require.Equal(t, 1, code)
	require.Contains(t, out, "! stub: 422 unprocessable")
}

func TestSettingsApplyExitsZeroOnSuccess(t *testing.T) {
	dir := settingsRepo(t, &stubGroup{changes: []settings.Change{
		{Key: "repository.has_wiki", From: "true", To: "false"},
	}})

	out, code := runSettings(t, dir)
	require.Equal(t, 0, code)
	require.Contains(t, out, "applied 1 setting")
}

func TestSettingsMissingFileIsAClearError(t *testing.T) {
	dir := t.TempDir()

	cmd := newSettingsCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"apply", "-C", dir})
	err := cmd.Execute()

	require.Error(t, err)
	require.Contains(t, err.Error(), settings.DefaultPath)
	var ec exitCodeError
	require.True(t, errors.As(err, &ec))
	require.Equal(t, 2, ec.code)
}

// TestSettingsNoLockRequiresRepoFlag covers the retrofit case: a repo keel never
// created has no lock, so the target has to come from the flag.
func TestSettingsNoLockRequiresRepoFlag(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".github"), 0o750))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, filepath.FromSlash(settings.DefaultPath)),
		[]byte("version: 1\nrepository:\n  has_wiki: false\n"), 0o600))

	cmd := newSettingsCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"apply", "-C", dir})
	err := cmd.Execute()

	require.Error(t, err)
	require.Contains(t, err.Error(), "--repo")
}

func TestSettingsRejectsMalformedRepoFlag(t *testing.T) {
	dir := settingsRepo(t, &stubGroup{})
	_, code := runSettings(t, dir, "--repo", "not-a-pair")
	require.Equal(t, 2, code)
}
