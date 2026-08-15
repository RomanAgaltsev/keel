package settings_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2/internal/settings"
)

// fakeGroup records what it was asked to do and returns canned results.
type fakeGroup struct {
	name        string
	changes     []settings.Change
	planErr     error
	applyErr    error
	applyCalled bool
	applied     []settings.Change
}

func (f *fakeGroup) Name() string { return f.name }

func (f *fakeGroup) Plan(_ context.Context, _ settings.Desired) ([]settings.Change, error) {
	return f.changes, f.planErr
}

func (f *fakeGroup) Apply(_ context.Context, ch []settings.Change) error {
	f.applyCalled = true
	f.applied = ch
	return f.applyErr
}

// unsupportingGroup also reports keys it cannot express.
type unsupportingGroup struct {
	fakeGroup
	unsupported []settings.Unsupported
}

func (u *unsupportingGroup) Unsupported(_ settings.Desired) []settings.Unsupported {
	return u.unsupported
}

func chg(key string) settings.Change {
	return settings.Change{Key: key, From: "false", To: "true"}
}

func TestReconcileCheckModeNeverApplies(t *testing.T) {
	g := &fakeGroup{name: "repository", changes: []settings.Change{chg("repository.has_wiki")}}

	rep := settings.Reconcile(context.Background(), []settings.Group{g}, settings.Desired{}, false)

	require.False(t, g.applyCalled, "check mode must not write")
	require.False(t, rep.Applied)
	require.False(t, rep.InSync())
	require.Len(t, rep.Changes, 1)
	require.Equal(t, "repository.has_wiki", rep.Changes[0].Key)
}

func TestReconcileNoChangesIssuesNoApply(t *testing.T) {
	g := &fakeGroup{name: "repository"}

	rep := settings.Reconcile(context.Background(), []settings.Group{g}, settings.Desired{}, true)

	require.False(t, g.applyCalled, "a group with no changes must not be applied")
	require.True(t, rep.InSync())
	require.Empty(t, rep.Failed)
}

func TestReconcileAppliesChanges(t *testing.T) {
	g := &fakeGroup{name: "actions", changes: []settings.Change{chg("actions.allowed")}}

	rep := settings.Reconcile(context.Background(), []settings.Group{g}, settings.Desired{}, true)

	require.True(t, g.applyCalled)
	require.Len(t, g.applied, 1)
	require.True(t, rep.Applied)
	require.Len(t, rep.Changes, 1)
	require.Empty(t, rep.Failed)
}

func TestReconcileContinuesAfterFailingGroup(t *testing.T) {
	bad := &fakeGroup{name: "security", planErr: errors.New("403 forbidden")}
	good := &fakeGroup{name: "repository", changes: []settings.Change{chg("repository.has_wiki")}}

	rep := settings.Reconcile(context.Background(), []settings.Group{bad, good}, settings.Desired{}, true)

	require.True(t, good.applyCalled, "a failing group must not stop later groups")
	require.Len(t, rep.Failed, 1)
	require.Equal(t, "security", rep.Failed[0].Group)
	require.Contains(t, rep.Failed[0].Reason, "403")
	require.Len(t, rep.Changes, 1)
}

func TestReconcileRecordsApplyFailure(t *testing.T) {
	g := &fakeGroup{name: "ruleset", changes: []settings.Change{chg("ruleset.main")}, applyErr: errors.New("422 unprocessable")}

	rep := settings.Reconcile(context.Background(), []settings.Group{g}, settings.Desired{}, true)

	require.Len(t, rep.Failed, 1)
	require.Equal(t, "ruleset", rep.Failed[0].Group)
	require.Contains(t, rep.Failed[0].Reason, "422")
}

func TestReconcileCollectsUnsupported(t *testing.T) {
	g := &unsupportingGroup{
		fakeGroup:   fakeGroup{name: "security"},
		unsupported: []settings.Unsupported{{Key: "security.dependency_graph", Provider: "bitbucket", Reason: "no equivalent concept"}},
	}

	rep := settings.Reconcile(context.Background(), []settings.Group{g}, settings.Desired{}, true)

	require.Len(t, rep.Unsupported, 1)
	require.Equal(t, "security.dependency_graph", rep.Unsupported[0].Key)
	require.True(t, rep.InSync(), "unsupported keys are not drift")
}
