package provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/provider"
)

func TestGitLabRepoExists(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.EscapedPath() {
		case "/projects/me%2Fexists":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"http_url_to_repo":"https://gl/me/exists.git","web_url":"https://gl/me/exists"}`))
		case "/projects/me%2Fmissing":
			w.WriteHeader(http.StatusNotFound)
		default:
			t.Fatalf("unexpected path %s", r.URL.EscapedPath())
		}
	}))
	defer srv.Close()

	gl := provider.NewGitLab("tok", "me", provider.WithGitLabBaseURL(srv.URL))

	exists, repo, err := gl.RepoExists(context.Background(), provider.RepoSpec{Name: "exists"})
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, "https://gl/me/exists.git", repo.CloneURL)

	exists, _, err = gl.RepoExists(context.Background(), provider.RepoSpec{Name: "missing"})
	require.NoError(t, err)
	require.False(t, exists)
}

func TestGitLabCreateRepo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/projects", r.URL.Path)
		require.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "new", body["name"])
		require.Equal(t, "private", body["visibility"])
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"http_url_to_repo":"https://gl/me/new.git","web_url":"https://gl/me/new"}`))
	}))
	defer srv.Close()

	gl := provider.NewGitLab("tok", "me", provider.WithGitLabBaseURL(srv.URL))
	repo, err := gl.CreateRepo(context.Background(), provider.RepoSpec{Name: "new", Private: true})
	require.NoError(t, err)
	require.Equal(t, "https://gl/me/new.git", repo.CloneURL)
}
