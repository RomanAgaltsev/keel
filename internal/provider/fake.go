package provider

import "context"

// Fake is an in-memory Provider for tests. Configure Exists/Repo/errors.
// Inspect Created after a run.
type Fake struct {
	Exists    bool
	Repo      RemoteRepo
	ExistsErr error
	CreateErr error

	Created bool // set true once CreateRepo is called
}

func (f *Fake) Name() string {
	return "fake"
}

func (f *Fake) RepoExists(_ context.Context, _ RepoSpec) (bool, RemoteRepo, error) {
	return f.Exists, f.Repo, f.ExistsErr
}

func (f *Fake) CreateRepo(_ context.Context, _ RepoSpec) (RemoteRepo, error) {
	if f.CreateErr != nil {
		return RemoteRepo{}, f.CreateErr
	}
	f.Created = true
	return f.Repo, nil
}
