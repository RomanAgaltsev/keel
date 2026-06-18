// Package provider abstracts remote git hosting (GitHub, GitLab,...).
package provider

import "context"

// RepoSpec describes the remote repository to create or look up.
type RepoSpec struct {
	Name        string
	Description string
	Private     bool
}

// RemoteRepo is a created or existing remote repository.
type RemoteRepo struct {
	CloneURL string // e.g. https://github.com/owner/name.git
	HTMLURL  string // e.g. https://github.com/owner/name
}

// Provider creates and inspects remote repositories on a hosting provider.
type Provider interface {
	// Name identifies the provider (e.g. "github").
	Name() string
	// RepoExists reports whether the remote already exists and, if so, its URLs,
	// so the orchestrator can choose clone-then-overlay instead of creating.
	RepoExists(ctx context.Context, spec RepoSpec) (bool, RemoteRepo, error)
	// CreateRepo creates the remote. Callers gate it on RepoExists.
	CreateRepo(ctx context.Context, spec RepoSpec) (RemoteRepo, error)
}
