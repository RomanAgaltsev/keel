package provider_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2/internal/provider"
	"github.com/RomanAgaltsev/keel/v2/internal/settings"
)

func ptr[T any](v T) *T { return &v }

// groupByName picks one group out of the provider's set so each test drives
// exactly one API surface.
func groupByName(t *testing.T, gh *provider.GitHub, name string) settings.Group {
	t.Helper()
	for _, g := range gh.SettingsGroups(provider.RepoSpec{Name: "demo"}) {
		if g.Name() == name {
			return g
		}
	}
	t.Fatalf("no group named %q", name)
	return nil
}

func TestGitHubImplementsSettingsApplier(t *testing.T) {
	var _ provider.SettingsApplier = (*provider.GitHub)(nil)
}

func TestRepositoryGroupPlansOnlyDeclaredDrift(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/repos/me/demo", r.URL.Path)
		_, _ = w.Write([]byte(`{"has_wiki":true,"has_projects":false,"allow_squash_merge":false,"delete_branch_on_merge":false}`))
	}))
	defer srv.Close()

	gh := provider.NewGitHub("tok", "me", provider.WithBaseURL(srv.URL))
	g := groupByName(t, gh, "repository")

	changes, err := g.Plan(context.Background(), settings.Desired{Repository: &settings.Repository{
		HasWiki:          ptr(false), // declared, drifted  -> change
		HasProjects:      ptr(false), // declared, matches  -> no change
		AllowSquashMerge: ptr(true),  // declared, drifted  -> change
		// delete_branch_on_merge undeclared -> untouched even though it is false
	}})
	require.NoError(t, err)

	keys := make([]string, 0, len(changes))
	for _, c := range changes {
		keys = append(keys, c.Key)
	}
	require.ElementsMatch(t, []string{"repository.has_wiki", "repository.allow_squash_merge"}, keys)
}

func TestRepositoryGroupNoDriftIssuesNoWrite(t *testing.T) {
	var writes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writes++
		}
		_, _ = w.Write([]byte(`{"has_wiki":false}`))
	}))
	defer srv.Close()

	gh := provider.NewGitHub("tok", "me", provider.WithBaseURL(srv.URL))
	g := groupByName(t, gh, "repository")

	changes, err := g.Plan(context.Background(), settings.Desired{Repository: &settings.Repository{HasWiki: ptr(false)}})
	require.NoError(t, err)
	require.Empty(t, changes, "matching state must produce no changes")
	require.Zero(t, writes, "no write request may be issued when nothing drifted")
}

func TestRepositoryGroupAppliesPatch(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"has_wiki":true}`))
			return
		}
		require.Equal(t, http.MethodPatch, r.Method)
		require.Equal(t, "/repos/me/demo", r.URL.Path)
		require.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
		b, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(b, &got))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	gh := provider.NewGitHub("tok", "me", provider.WithBaseURL(srv.URL))
	g := groupByName(t, gh, "repository")

	d := settings.Desired{Repository: &settings.Repository{HasWiki: ptr(false)}}
	changes, err := g.Plan(context.Background(), d)
	require.NoError(t, err)
	require.NoError(t, g.Apply(context.Background(), changes))

	require.Equal(t, map[string]any{"has_wiki": false}, got,
		"the PATCH body must carry only the drifted, declared fields")
}

func TestRepositoryGroupSkippedWhenSectionAbsent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("no request may be made when the repository section is undeclared")
	}))
	defer srv.Close()

	gh := provider.NewGitHub("tok", "me", provider.WithBaseURL(srv.URL))
	changes, err := groupByName(t, gh, "repository").Plan(context.Background(), settings.Desired{})
	require.NoError(t, err)
	require.Empty(t, changes)
}

func TestTopicsGroupReplacesWholeList(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/me/demo/topics", r.URL.Path)
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"names":["old"]}`))
			return
		}
		require.Equal(t, http.MethodPut, r.Method)
		b, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(b, &got))
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	gh := provider.NewGitHub("tok", "me", provider.WithBaseURL(srv.URL))
	g := groupByName(t, gh, "topics")

	d := settings.Desired{Repository: &settings.Repository{Topics: ptr([]string{"go", "cli"})}}
	changes, err := g.Plan(context.Background(), d)
	require.NoError(t, err)
	require.Len(t, changes, 1)
	require.Equal(t, "repository.topics", changes[0].Key)
	require.Equal(t, "[old]", changes[0].From)
	require.Equal(t, "[go cli]", changes[0].To)

	require.NoError(t, g.Apply(context.Background(), changes))
	require.Equal(t, map[string]any{"names": []any{"go", "cli"}}, got)
}

func TestGroupSurfacesAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"Resource not accessible by personal access token"}`))
	}))
	defer srv.Close()

	gh := provider.NewGitHub("tok", "me", provider.WithBaseURL(srv.URL))
	_, err := groupByName(t, gh, "repository").Plan(context.Background(),
		settings.Desired{Repository: &settings.Repository{HasWiki: ptr(false)}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "403")
}
