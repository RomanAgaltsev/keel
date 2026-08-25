package godeps_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2/internal/godeps"
)

func TestEscapePathLowercasesCapitals(t *testing.T) {
	// The proxy requires capital letters to be escaped as !<lowercase>, and a
	// request with an unescaped capital 404s. This is a real source of silent
	// misses, so it is tested directly rather than only through the client.
	require.Equal(t, "github.com/!burnt!sushi/toml", godeps.EscapePath("github.com/BurntSushi/toml"))
	require.Equal(t, "golang.org/x/tools", godeps.EscapePath("golang.org/x/tools"))
}

func TestLatestVersionReadsTheProxy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/golang.org/x/tools/@latest", r.URL.Path)
		_, _ = w.Write([]byte(`{"Version":"v0.38.0","Time":"2026-08-01T00:00:00Z"}`))
	}))
	defer srv.Close()

	got, err := godeps.NewProxy(godeps.WithBaseURL(srv.URL)).
		LatestVersion(context.Background(), "golang.org/x/tools")
	require.NoError(t, err)
	require.Equal(t, "v0.38.0", got)
}

func TestLatestVersionEscapesTheRequestPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/github.com/!burnt!sushi/toml/@latest", r.URL.Path)
		_, _ = w.Write([]byte(`{"Version":"v1.5.0"}`))
	}))
	defer srv.Close()

	got, err := godeps.NewProxy(godeps.WithBaseURL(srv.URL)).
		LatestVersion(context.Background(), "github.com/BurntSushi/toml")
	require.NoError(t, err)
	require.Equal(t, "v1.5.0", got)
}

func TestLatestVersionErrorsOnNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := godeps.NewProxy(godeps.WithBaseURL(srv.URL)).
		LatestVersion(context.Background(), "example.com/nope")
	require.Error(t, err)
}
