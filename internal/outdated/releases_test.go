package outdated_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/outdated"
)

func TestLatestTag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/actions/checkout/releases/latest":
			_, _ = w.Write([]byte(`{"tag_name":"v5.2.1"}`))
		case "/repos/some/norelease/releases/latest":
			w.WriteHeader(http.StatusNotFound)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	rc := outdated.NewGitHubReleases("", outdated.WithReleasesBaseURL(srv.URL))

	tag, err := rc.LatestTag(context.Background(), "actions/checkout")
	require.NoError(t, err)
	require.Equal(t, "v5.2.1", tag)

	_, err = rc.LatestTag(context.Background(), "some/norelease")
	require.ErrorIs(t, err, outdated.ErrNoRelease)
}
