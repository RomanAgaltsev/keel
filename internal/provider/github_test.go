package provider_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/provider"
)

func TestGitHubRepoExists(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/me/exists":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"clone_url":"https://h/me/exists.git","html_url":"https://h/me/exists"}`))
		case "/repos/me/missing":
			w.WriteHeader(http.StatusNotFound)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	gh := provider.NewGitHub("tok", "me", provider.WithBaseURL(srv.URL))

	exists, repo, err := gh.RepoExists(context.Background(), provider.RepoSpec{Name: "exists"})
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, "https://h/me/exists.git", repo.CloneURL)

	exists, _, err = gh.RepoExists(context.Background(), provider.RepoSpec{Name: "missing"})
	require.NoError(t, err)
	require.False(t, exists)
}

func TestGitHubCreateRepoUnderTheTokenUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/user":
			_, _ = w.Write([]byte(`{"login":"me"}`))
		case "/user/repos":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"clone_url":"https://h/me/new.git","html_url":"https://h/me/new"}`))
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	}))
	defer srv.Close()

	gh := provider.NewGitHub("tok", "me", provider.WithBaseURL(srv.URL))
	repo, err := gh.CreateRepo(context.Background(), provider.RepoSpec{Name: "new", Private: true})
	require.NoError(t, err)
	require.Equal(t, "https://h/me/new.git", repo.CloneURL)
}

// TestGitHubCreateRepoUnderAnOrg is the regression test for the bug this fixes:
// creating with owner set to an organization posted to /user/repos, so the repo
// landed in the token user's namespace while the existence check kept looking at
// the org — reporting "already exists" on the second run for a repo the check
// had just said was missing.
func TestGitHubCreateRepoUnderAnOrg(t *testing.T) {
	var createPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			_, _ = w.Write([]byte(`{"login":"token-user"}`))
		default:
			createPath = r.URL.Path
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"clone_url":"https://h/my-org/widget.git","html_url":"https://h/my-org/widget"}`))
		}
	}))
	defer srv.Close()

	gh := provider.NewGitHub("tok", "my-org", provider.WithBaseURL(srv.URL))
	repo, err := gh.CreateRepo(context.Background(), provider.RepoSpec{Name: "widget"})
	require.NoError(t, err)
	require.Equal(t, "/orgs/my-org/repos", createPath)
	require.Equal(t, "https://h/my-org/widget.git", repo.CloneURL)
}

// TestGitHubCreateRepoOwnerMatchIsCaseInsensitive guards the comparison itself:
// GitHub logins are case-insensitive, so "Me" and "me" are the same namespace and
// must not be mistaken for an org.
func TestGitHubCreateRepoOwnerMatchIsCaseInsensitive(t *testing.T) {
	var createPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			_, _ = w.Write([]byte(`{"login":"Me"}`))
		default:
			createPath = r.URL.Path
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"clone_url":"https://h/Me/new.git","html_url":"https://h/Me/new"}`))
		}
	}))
	defer srv.Close()

	gh := provider.NewGitHub("tok", "me", provider.WithBaseURL(srv.URL))
	_, err := gh.CreateRepo(context.Background(), provider.RepoSpec{Name: "new"})
	require.NoError(t, err)
	require.Equal(t, "/user/repos", createPath)
}

// TestGitHubCreateRepoFallsBackWhenUserLookupFails keeps a create working against
// a token whose /user call is unavailable: the previous behaviour is still right
// for the common personal-account case.
func TestGitHubCreateRepoFallsBackWhenUserLookupFails(t *testing.T) {
	var createPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			w.WriteHeader(http.StatusForbidden)
		default:
			createPath = r.URL.Path
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"clone_url":"https://h/me/new.git","html_url":"https://h/me/new"}`))
		}
	}))
	defer srv.Close()

	gh := provider.NewGitHub("tok", "me", provider.WithBaseURL(srv.URL))
	_, err := gh.CreateRepo(context.Background(), provider.RepoSpec{Name: "new"})
	require.NoError(t, err)
	require.Equal(t, "/user/repos", createPath)
}

// TestGitHubCreateRepoErrors drives CreateRepo through non-success responses to
// exercise apiError's three branches: already-exists conflict, body message,
// and empty body.
func TestGitHubCreateRepoErrors(t *testing.T) {
	t.Run("already exists maps to ErrRepoExists", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"message":"Repository creation failed: name already exists on this account"}`))
		}))
		defer srv.Close()

		gh := provider.NewGitHub("tok", "me", provider.WithBaseURL(srv.URL))
		_, err := gh.CreateRepo(context.Background(), provider.RepoSpec{Name: "dup"})
		require.ErrorIs(t, err, provider.ErrRepoExists)
	})

	t.Run("body message is surfaced", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
		}))
		defer srv.Close()

		gh := provider.NewGitHub("tok", "me", provider.WithBaseURL(srv.URL))
		_, err := gh.CreateRepo(context.Background(), provider.RepoSpec{Name: "x"})
		require.Error(t, err)
		require.NotErrorIs(t, err, provider.ErrRepoExists)
		require.ErrorContains(t, err, "status 500")
		require.ErrorContains(t, err, "boom")
	})

	t.Run("empty body falls back to status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer srv.Close()

		gh := provider.NewGitHub("tok", "me", provider.WithBaseURL(srv.URL))
		_, err := gh.CreateRepo(context.Background(), provider.RepoSpec{Name: "x"})
		require.ErrorContains(t, err, "status 401")
	})
}
