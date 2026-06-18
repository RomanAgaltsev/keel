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
