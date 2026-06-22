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

func TestGitHubCreateRepo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/user/repos", r.URL.Path)
		require.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"clone_url":"https://h/me/new.git","html_url":"https://h/me/new"}`))
	}))
	defer srv.Close()

	gh := provider.NewGitHub("tok", "me", provider.WithBaseURL(srv.URL))
	repo, err := gh.CreateRepo(context.Background(), provider.RepoSpec{Name: "new", Private: true})
	require.NoError(t, err)
	require.Equal(t, "https://h/me/new.git", repo.CloneURL)
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
