package provider_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/provider"
)

const bbExistsBody = `{"links":{"html":{"href":"https://bitbucket.org/ws/exists"},"clone":[{"name":"https","href":"https://bitbucket.org/ws/exists.git"},{"name":"ssh","href":"git@bitbucket.org:ws/exists.git"}]}}`

func TestBitbucketRepoExists(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repositories/ws/exists":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(bbExistsBody))
		case "/repositories/ws/missing":
			w.WriteHeader(http.StatusNotFound)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	bb := provider.NewBitbucket("tok", "ws", provider.WithBitbucketBaseURL(srv.URL))

	exists, repo, err := bb.RepoExists(context.Background(), provider.RepoSpec{Name: "exists"})
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, "https://bitbucket.org/ws/exists.git", repo.CloneURL)

	exists, _, err = bb.RepoExists(context.Background(), provider.RepoSpec{Name: "missing"})
	require.NoError(t, err)
	require.False(t, exists)
}

func TestBitbucketCreateRepo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repositories/ws/new", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"links":{"html":{"href":"https://bitbucket.org/ws/new"},"clone":[{"name":"https","href":"https://bitbucket.org/ws/new.git"}]}}`))
	}))
	defer srv.Close()

	bb := provider.NewBitbucket("tok", "ws", provider.WithBitbucketBaseURL(srv.URL))
	repo, err := bb.CreateRepo(context.Background(), provider.RepoSpec{Name: "new", Private: true})
	require.NoError(t, err)
	require.Equal(t, "https://bitbucket.org/ws/new.git", repo.CloneURL)
}
