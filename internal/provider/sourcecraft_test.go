package provider_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/provider"
)

func TestSourceCraftRepoExists(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/myorg/exists": // single-repo GET path is /repos/{org_slug}/{repo_slug}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"clone_url":{"https":"https://git.sourcecraft.dev/myorg/exists.git","ssh":"git@git.sourcecraft.dev:myorg/exists.git"},"web_url":"https://sourcecraft.dev/myorg/exists"}`)) // Task 1: clone_url is a nested {https,ssh} object
		case "/repos/myorg/missing":
			w.WriteHeader(http.StatusNotFound)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	sc := provider.NewSourceCraft("tok", "myorg", provider.WithSourceCraftBaseURL(srv.URL))

	exists, repo, err := sc.RepoExists(context.Background(), provider.RepoSpec{Name: "exists"})
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, "https://git.sourcecraft.dev/myorg/exists.git", repo.CloneURL)

	exists, _, err = sc.RepoExists(context.Background(), provider.RepoSpec{Name: "missing"})
	require.NoError(t, err)
	require.False(t, exists)
}

func TestSourceCraftCreateRepo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/orgs/myorg/repos", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusCreated) // Task 1: create returns 201
		_, _ = w.Write([]byte(`{"clone_url":{"https":"https://git.sourcecraft.dev/myorg/new.git","ssh":"git@git.sourcecraft.dev:myorg/new.git"},"web_url":"https://sourcecraft.dev/myorg/new"}`))
	}))
	defer srv.Close()

	sc := provider.NewSourceCraft("tok", "myorg", provider.WithSourceCraftBaseURL(srv.URL))
	repo, err := sc.CreateRepo(context.Background(), provider.RepoSpec{Name: "new", Private: true})
	require.NoError(t, err)
	require.Equal(t, "https://git.sourcecraft.dev/myorg/new.git", repo.CloneURL)
}
