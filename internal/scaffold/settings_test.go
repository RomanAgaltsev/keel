package scaffold_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2/internal/provider"
	"github.com/RomanAgaltsev/keel/v2/internal/scaffold"
	"github.com/RomanAgaltsev/keel/v2/internal/settings"
)

// recordingGroup is a settings.Group that records whether it was applied.
type recordingGroup struct {
	changes []settings.Change
	applied bool
	err     error
}

func (r *recordingGroup) Name() string { return "recording" }

func (r *recordingGroup) Plan(_ context.Context, _ settings.Desired) ([]settings.Change, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.changes, nil
}

func (r *recordingGroup) Apply(_ context.Context) error {
	r.applied = true
	return nil
}

// writeSettingsFile drops a minimal valid settings file into dir.
func writeSettingsFile(t *testing.T, dir string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(settings.DefaultPath))
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o750))
	require.NoError(t, os.WriteFile(p, []byte("version: 1\nrepository:\n  has_wiki: false\n"), 0o600))
}

// settingsOpts builds a scaffold that will reach the settings step: an existing
// local tree (so nothing is cloned) carrying a settings file, and a provider that
// reports the remote as already present.
func settingsOpts(t *testing.T, p provider.Provider) scaffold.Options {
	t.Helper()
	target := filepath.Join(t.TempDir(), "demo")
	require.NoError(t, os.MkdirAll(target, 0o750))
	writeSettingsFile(t, target)

	opts := baseOpts(target, p)
	opts.CreateRemote = true
	return opts
}

func TestSettingsAppliedAfterRemoteStep(t *testing.T) {
	g := &recordingGroup{changes: []settings.Change{{Key: "repository.has_wiki", From: "true", To: "false"}}}
	f := &provider.FakeApplier{
		Fake:   provider.Fake{Exists: true, Repo: provider.RemoteRepo{CloneURL: emptyBare(t)}},
		Groups: []settings.Group{g},
	}

	res, err := scaffold.Run(context.Background(), settingsOpts(t, f))
	require.NoError(t, err)

	require.NotNil(t, res.Settings)
	require.True(t, g.applied, "the settings step must apply, not merely plan")
	require.Empty(t, res.Settings.Failed)
	require.Len(t, res.Settings.Changes, 1)
}

func TestSettingsAbsentFileIsSilent(t *testing.T) {
	g := &recordingGroup{}
	f := &provider.FakeApplier{
		Fake:   provider.Fake{Exists: true, Repo: provider.RemoteRepo{CloneURL: emptyBare(t)}},
		Groups: []settings.Group{g},
	}
	opts := settingsOpts(t, f)
	require.NoError(t, os.Remove(filepath.Join(opts.Target, filepath.FromSlash(settings.DefaultPath))))

	res, err := scaffold.Run(context.Background(), opts)
	require.NoError(t, err, "a repo with no settings file scaffolds normally")
	require.Nil(t, res.Settings)
	require.False(t, g.applied)
}

// TestSettingsUnsupportedProviderIsReported covers the capability gap: a provider
// that cannot reconcile settings must say so rather than fail or stay silent.
func TestSettingsUnsupportedProviderIsReported(t *testing.T) {
	f := &provider.Fake{Exists: true, Repo: provider.RemoteRepo{CloneURL: emptyBare(t)}}

	res, err := scaffold.Run(context.Background(), settingsOpts(t, f))
	require.NoError(t, err)
	require.Nil(t, res.Settings)
	require.Contains(t, strings.Join(res.NextSteps, "\n"), "not supported for provider fake")
}

// TestSettingsFailureIsNotFatalAndSuggestsRetry is the load-bearing guarantee of
// this task: by the time settings run the repo exists and the code is pushed, so
// failing the scaffold could not undo anything — it would only leave the user
// with a working repo and an error exit.
func TestSettingsFailureIsNotFatalAndSuggestsRetry(t *testing.T) {
	g := &recordingGroup{err: errors.New("403 forbidden")}
	f := &provider.FakeApplier{
		Fake:   provider.Fake{Exists: true, Repo: provider.RemoteRepo{CloneURL: emptyBare(t)}},
		Groups: []settings.Group{g},
	}

	res, err := scaffold.Run(context.Background(), settingsOpts(t, f))
	require.NoError(t, err, "a settings failure must never fail the scaffold")
	require.NotNil(t, res.Settings)
	require.Len(t, res.Settings.Failed, 1)
	require.Contains(t, res.Settings.Failed[0].Reason, "403")
	require.Contains(t, strings.Join(res.NextSteps, "\n"), "keel settings apply")
}

// TestSettingsSkippedWithoutRemote keeps keel from reading a settings file when
// there is no remote to apply it to (--no-remote, or a local-only scaffold).
func TestSettingsSkippedWithoutRemote(t *testing.T) {
	g := &recordingGroup{}
	f := &provider.FakeApplier{Fake: provider.Fake{}, Groups: []settings.Group{g}}
	opts := settingsOpts(t, f)
	opts.CreateRemote = false

	res, err := scaffold.Run(context.Background(), opts)
	require.NoError(t, err)
	require.Nil(t, res.Settings)
	require.False(t, g.applied)
}
