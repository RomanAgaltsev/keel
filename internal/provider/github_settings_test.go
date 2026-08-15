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

func TestSecurityGroupPatchesAnalysisAndTogglesAlerts(t *testing.T) {
	var patch map[string]any
	var puts []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/me/demo":
			_, _ = w.Write([]byte(`{"security_and_analysis":{"secret_scanning":{"status":"disabled"},"secret_scanning_push_protection":{"status":"disabled"},"dependency_graph":{"status":"enabled"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/repos/me/demo/vulnerability-alerts":
			w.WriteHeader(http.StatusNotFound) // alerts currently off
		case r.Method == http.MethodGet && r.URL.Path == "/repos/me/demo/automated-security-fixes":
			w.WriteHeader(http.StatusNoContent) // security updates currently on
		case r.Method == http.MethodPatch:
			b, _ := io.ReadAll(r.Body)
			require.NoError(t, json.Unmarshal(b, &patch))
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodPut:
			puts = append(puts, r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	gh := provider.NewGitHub("tok", "me", provider.WithBaseURL(srv.URL))
	g := groupByName(t, gh, "security")

	d := settings.Desired{Security: &settings.Security{
		SecretScanning:            ptr(true), // drifted
		DependencyGraph:           ptr(true), // matches -> no change
		DependabotAlerts:          ptr(true), // drifted (404 -> on)
		DependabotSecurityUpdates: ptr(true), // matches (204) -> no change
	}}
	changes, err := g.Plan(context.Background(), d)
	require.NoError(t, err)
	require.NoError(t, g.Apply(context.Background(), changes))

	keys := make([]string, 0, len(changes))
	for _, c := range changes {
		keys = append(keys, c.Key)
	}
	require.ElementsMatch(t, []string{"security.secret_scanning", "security.dependabot_alerts"}, keys)

	sa, ok := patch["security_and_analysis"].(map[string]any)
	require.True(t, ok, "PATCH must carry a security_and_analysis object")
	require.Equal(t, map[string]any{"status": "enabled"}, sa["secret_scanning"])
	require.NotContains(t, sa, "dependency_graph", "a matching key must not be written")
	require.Equal(t, []string{"/repos/me/demo/vulnerability-alerts"}, puts)
}

// TestSecurityGroupReadsToggleFrom200WithBody pins the shape verified live on
// 2026-08-15: automated-security-fixes answers 200 with {"enabled":...} while
// vulnerability-alerts answers a bodiless 204. Both mean "on", and both answer
// 404 when off, so the reader is status-only and must accept 200 as well as 204.
func TestSecurityGroupReadsToggleFrom200WithBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/me/demo/automated-security-fixes" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"enabled":true,"paused":false}`))
			return
		}
		t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	gh := provider.NewGitHub("tok", "me", provider.WithBaseURL(srv.URL))
	changes, err := groupByName(t, gh, "security").Plan(context.Background(), settings.Desired{
		Security: &settings.Security{DependabotSecurityUpdates: ptr(true)},
	})
	require.NoError(t, err)
	require.Empty(t, changes, "200-with-body must read as enabled, not as drift")
}

func TestSecurityGroupDisablesWithDelete(t *testing.T) {
	var deletes []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/me/demo":
			_, _ = w.Write([]byte(`{"security_and_analysis":{}}`))
		case r.Method == http.MethodGet:
			w.WriteHeader(http.StatusNoContent) // currently enabled
		case r.Method == http.MethodDelete:
			deletes = append(deletes, r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	gh := provider.NewGitHub("tok", "me", provider.WithBaseURL(srv.URL))
	g := groupByName(t, gh, "security")

	changes, err := g.Plan(context.Background(), settings.Desired{
		Security: &settings.Security{DependabotAlerts: ptr(false)},
	})
	require.NoError(t, err)
	require.Len(t, changes, 1)
	require.NoError(t, g.Apply(context.Background(), changes))
	require.Equal(t, []string{"/repos/me/demo/vulnerability-alerts"}, deletes)
}

func TestVulnReportingGroup(t *testing.T) {
	var puts []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/me/demo/private-vulnerability-reporting", r.URL.Path)
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		puts = append(puts, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	gh := provider.NewGitHub("tok", "me", provider.WithBaseURL(srv.URL))
	g := groupByName(t, gh, "vuln-reporting")

	changes, err := g.Plan(context.Background(), settings.Desired{
		Security: &settings.Security{PrivateVulnerabilityReporting: ptr(true)},
	})
	require.NoError(t, err)
	require.Len(t, changes, 1)
	require.Equal(t, "security.private_vulnerability_reporting", changes[0].Key)
	require.NoError(t, g.Apply(context.Background(), changes))
	require.Equal(t, []string{http.MethodPut}, puts)
}

func TestSecurityGroupSkippedWhenSectionAbsent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("no request may be made when the security section is undeclared")
	}))
	defer srv.Close()

	gh := provider.NewGitHub("tok", "me", provider.WithBaseURL(srv.URL))
	for _, name := range []string{"security", "vuln-reporting"} {
		changes, err := groupByName(t, gh, name).Plan(context.Background(), settings.Desired{})
		require.NoError(t, err)
		require.Empty(t, changes)
	}
}

func TestActionsGroupTranslatesLocalAndVerified(t *testing.T) {
	bodies := map[string]map[string]any{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/me/demo/actions/permissions":
			_, _ = w.Write([]byte(`{"enabled":true,"allowed_actions":"all"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/repos/me/demo/actions/permissions/workflow":
			_, _ = w.Write([]byte(`{"default_workflow_permissions":"write","can_approve_pull_request_reviews":true}`))
		case r.Method == http.MethodPut:
			var b map[string]any
			raw, _ := io.ReadAll(r.Body)
			require.NoError(t, json.Unmarshal(raw, &b))
			bodies[r.URL.Path] = b
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	gh := provider.NewGitHub("tok", "me", provider.WithBaseURL(srv.URL))
	g := groupByName(t, gh, "actions")

	d := settings.Desired{Actions: &settings.Actions{
		Allowed:                      ptr(settings.AllowedLocalAndVerified),
		DefaultWorkflowPermissions:   ptr("read"),
		CanApprovePullRequestReviews: ptr(false),
	}}
	changes, err := g.Plan(context.Background(), d)
	require.NoError(t, err)
	require.NoError(t, g.Apply(context.Background(), changes))

	keys := make([]string, 0, len(changes))
	for _, c := range changes {
		keys = append(keys, c.Key)
	}
	require.ElementsMatch(t, []string{
		"actions.allowed",
		"actions.default_workflow_permissions",
		"actions.can_approve_pull_request_reviews",
	}, keys)

	require.Equal(t, "selected", bodies["/repos/me/demo/actions/permissions"]["allowed_actions"],
		"local_and_verified must translate to allowed_actions=selected")
	sel := bodies["/repos/me/demo/actions/permissions/selected-actions"]
	require.Equal(t, true, sel["github_owned_allowed"])
	require.Equal(t, true, sel["verified_allowed"])
	wf := bodies["/repos/me/demo/actions/permissions/workflow"]
	require.Equal(t, "read", wf["default_workflow_permissions"])
	require.Equal(t, false, wf["can_approve_pull_request_reviews"])
}

// TestActionsGroupNeverReadsSelectedActionsUnlessSelected pins a live finding
// from 2026-08-15: GET …/selected-actions answers 409 Conflict — not 404, not
// defaults — while allowed_actions is anything but "selected". Reading it
// unconditionally would turn every "all" repo into a spurious group failure.
func TestActionsGroupNeverReadsSelectedActionsUnlessSelected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/me/demo/actions/permissions/selected-actions" {
			w.WriteHeader(http.StatusConflict)
			return
		}
		_, _ = w.Write([]byte(`{"enabled":true,"allowed_actions":"all"}`))
	}))
	defer srv.Close()

	gh := provider.NewGitHub("tok", "me", provider.WithBaseURL(srv.URL))
	changes, err := groupByName(t, gh, "actions").Plan(context.Background(), settings.Desired{
		Actions: &settings.Actions{Allowed: ptr(settings.AllowedLocalOnly)},
	})
	require.NoError(t, err, "a 409 from selected-actions must never be reached")
	require.Len(t, changes, 1)
}

func TestActionsGroupReadsSelectedActionsWhenSelected(t *testing.T) {
	var writes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writes++
			w.WriteHeader(http.StatusNoContent)
			return
		}
		switch r.URL.Path {
		case "/repos/me/demo/actions/permissions":
			_, _ = w.Write([]byte(`{"enabled":true,"allowed_actions":"selected"}`))
		case "/repos/me/demo/actions/permissions/selected-actions":
			_, _ = w.Write([]byte(`{"github_owned_allowed":true,"verified_allowed":true,"patterns_allowed":[]}`))
		default:
			t.Fatalf("unexpected GET %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	gh := provider.NewGitHub("tok", "me", provider.WithBaseURL(srv.URL))
	changes, err := groupByName(t, gh, "actions").Plan(context.Background(), settings.Desired{
		Actions: &settings.Actions{Allowed: ptr(settings.AllowedLocalAndVerified)},
	})
	require.NoError(t, err)
	require.Empty(t, changes, "already local+verified must read as in sync")
	require.Zero(t, writes)
}

func TestActionsGroupSkippedWhenSectionAbsent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("no request may be made when the actions section is undeclared")
	}))
	defer srv.Close()

	gh := provider.NewGitHub("tok", "me", provider.WithBaseURL(srv.URL))
	changes, err := groupByName(t, gh, "actions").Plan(context.Background(), settings.Desired{})
	require.NoError(t, err)
	require.Empty(t, changes)
}
