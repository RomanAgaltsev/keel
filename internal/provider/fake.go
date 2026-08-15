package provider

import (
	"context"

	"github.com/RomanAgaltsev/keel/v2/internal/settings"
)

// Fake is an in-memory Provider for tests. Configure Exists/Repo/errors.
// Inspect Created after a run.
type Fake struct {
	Exists    bool
	Repo      RemoteRepo
	ExistsErr error
	CreateErr error

	Created      bool // set true once CreateRepo is called
	ExistsCalled bool // set true once RepoExists is called (e.g. to assert no network on dry-run)
}

func (f *Fake) Name() string {
	return "fake"
}

func (f *Fake) RepoExists(_ context.Context, _ RepoSpec) (bool, RemoteRepo, error) {
	f.ExistsCalled = true
	return f.Exists, f.Repo, f.ExistsErr
}

func (f *Fake) CreateRepo(_ context.Context, _ RepoSpec) (RemoteRepo, error) {
	if f.CreateErr != nil {
		return RemoteRepo{}, f.CreateErr
	}
	f.Created = true
	return f.Repo, nil
}

// FakeApplier is a Fake that also implements SettingsApplier, for tests that
// exercise the settings step. Kept as a distinct type rather than a flag on Fake
// so that a test wanting an *unsupported* provider simply uses plain *Fake and
// the type assertion in scaffold does the rest.
type FakeApplier struct {
	Fake
	Groups []settings.Group
}

// SettingsGroups returns the configured groups, ignoring the spec.
func (f *FakeApplier) SettingsGroups(_ RepoSpec) []settings.Group { return f.Groups }
