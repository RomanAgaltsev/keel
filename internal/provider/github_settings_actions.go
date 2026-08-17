package provider

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/RomanAgaltsev/keel/v2/internal/settings"
)

// diffString is diffBool for string-valued fields.
func diffString(out *[]settings.Change, body map[string]any, key, field, got string, want *string) {
	if want == nil || *want == got {
		return
	}
	body[field] = *want
	*out = append(*out, settings.Change{Key: key, From: got, To: *want})
}

// ghActionsGroup reconciles the Actions policy. It spans three endpoints because
// keel's local_and_verified has no single GitHub enum value: it is
// allowed_actions=selected plus a selected-actions payload.
type ghActionsGroup struct {
	gh       *GitHub
	permURL  string
	enabled  bool           // current …/actions/permissions "enabled", echoed back on write
	perm     map[string]any // staged PUT …/actions/permissions
	selected map[string]any // staged PUT …/actions/permissions/selected-actions
	workflow map[string]any // staged PUT …/actions/permissions/workflow
}

func (a *ghActionsGroup) Name() string { return "actions" }

func (a *ghActionsGroup) Plan(ctx context.Context, d settings.Desired) ([]settings.Change, error) {
	if d.Actions == nil {
		return nil, nil
	}
	a.perm, a.selected, a.workflow = nil, nil, nil
	var out []settings.Change
	if err := a.planAllowed(ctx, &out, d.Actions); err != nil {
		return nil, err
	}
	if err := a.planWorkflow(ctx, &out, d.Actions); err != nil {
		return nil, err
	}
	return out, nil
}

// planAllowed reads the effective policy and stages the writes for both the
// enum and the pattern list, which drift independently: a repo can sit on the
// right policy with the wrong allow-list, and before these were split that was
// invisible to `settings apply --check`.
//
// The selected-actions read is deliberately guarded inside readAllowed:
// verified live on 2026-08-15, that endpoint answers 409 Conflict (not 404, not
// defaults) unless allowed_actions is already "selected", so reading it
// unconditionally would turn every ordinary repo into a group failure.
func (a *ghActionsGroup) planAllowed(ctx context.Context, out *[]settings.Change, d *settings.Actions) error {
	if d.Allowed == nil {
		return nil
	}
	got, gotPatterns, err := a.readAllowed(ctx)
	if err != nil {
		return err
	}

	wantSelected := *d.Allowed == settings.AllowedLocalAndVerified
	patternsDiffer := wantSelected && !sameSet(gotPatterns, d.AllowedPatterns)
	if got == *d.Allowed && !patternsDiffer {
		return nil
	}

	// "enabled" is required by the endpoint — omitting it is a 422, caught by the
	// 2026-08-15 acceptance run — and the current value is echoed back rather than
	// assumed, so setting a policy never switches Actions on or off as a side effect.
	if got != *d.Allowed {
		allowed := *d.Allowed
		if wantSelected {
			allowed = "selected"
		}
		a.perm = map[string]any{"enabled": a.enabled, "allowed_actions": allowed}
		*out = append(*out, settings.Change{Key: "actions.allowed", From: got, To: *d.Allowed})
	}

	if wantSelected {
		pats := d.AllowedPatterns
		if pats == nil {
			pats = []string{}
		}
		a.selected = map[string]any{
			"github_owned_allowed": true,
			"verified_allowed":     true,
			"patterns_allowed":     pats,
		}
		if patternsDiffer {
			*out = append(*out, settings.Change{
				Key:  "actions.allowed_patterns",
				From: strings.Join(gotPatterns, ","),
				To:   strings.Join(pats, ","),
			})
		}
	}
	return nil
}

// readAllowed returns the effective policy in keel's vocabulary, resolving
// "selected" into local_and_verified when the selected-actions payload says so,
// along with the patterns currently allowed.
func (a *ghActionsGroup) readAllowed(ctx context.Context) (string, []string, error) {
	var cur struct {
		Enabled        bool   `json:"enabled"`
		AllowedActions string `json:"allowed_actions"`
	}
	if err := a.gh.getJSON(ctx, a.permURL, &cur); err != nil {
		return "", nil, err
	}
	a.enabled = cur.Enabled
	if cur.AllowedActions != "selected" {
		return cur.AllowedActions, nil, nil
	}
	var sel struct {
		GitHubOwnedAllowed bool     `json:"github_owned_allowed"`
		VerifiedAllowed    bool     `json:"verified_allowed"`
		PatternsAllowed    []string `json:"patterns_allowed"`
	}
	if err := a.gh.getJSON(ctx, a.permURL+"/selected-actions", &sel); err != nil {
		return "", nil, err
	}
	if sel.GitHubOwnedAllowed && sel.VerifiedAllowed {
		return settings.AllowedLocalAndVerified, sel.PatternsAllowed, nil
	}
	return cur.AllowedActions, sel.PatternsAllowed, nil
}

// sameSet compares two pattern lists ignoring order and duplicates.
func sameSet(a, b []string) bool {
	as, bs := slices.Clone(a), slices.Clone(b)
	slices.Sort(as)
	slices.Sort(bs)
	return slices.Equal(slices.Compact(as), slices.Compact(bs))
}

// planWorkflow reconciles the default job-token permissions.
func (a *ghActionsGroup) planWorkflow(ctx context.Context, out *[]settings.Change, w *settings.Actions) error {
	if w.DefaultWorkflowPermissions == nil && w.CanApprovePullRequestReviews == nil {
		return nil
	}
	var cur struct {
		DefaultWorkflowPermissions   string `json:"default_workflow_permissions"`
		CanApprovePullRequestReviews bool   `json:"can_approve_pull_request_reviews"`
	}
	url := a.permURL + "/workflow"
	if err := a.gh.getJSON(ctx, url, &cur); err != nil {
		return err
	}
	body := map[string]any{}
	diffString(out, body, "actions.default_workflow_permissions", "default_workflow_permissions", cur.DefaultWorkflowPermissions, w.DefaultWorkflowPermissions)
	diffBool(out, body, "actions.can_approve_pull_request_reviews", "can_approve_pull_request_reviews", cur.CanApprovePullRequestReviews, w.CanApprovePullRequestReviews)
	if len(body) == 0 {
		return nil
	}
	// The endpoint replaces both fields, so send the current value for whichever
	// one is undeclared rather than letting GitHub reset it.
	if _, ok := body["default_workflow_permissions"]; !ok {
		body["default_workflow_permissions"] = cur.DefaultWorkflowPermissions
	}
	if _, ok := body["can_approve_pull_request_reviews"]; !ok {
		body["can_approve_pull_request_reviews"] = cur.CanApprovePullRequestReviews
	}
	a.workflow = body
	return nil
}

func (a *ghActionsGroup) Apply(ctx context.Context) error {
	if a.perm != nil {
		if err := a.gh.sendJSON(ctx, http.MethodPut, a.permURL, a.perm); err != nil {
			return err
		}
	}
	if a.selected != nil {
		if err := a.gh.sendJSON(ctx, http.MethodPut, a.permURL+"/selected-actions", a.selected); err != nil {
			return err
		}
	}
	if a.workflow != nil {
		if err := a.gh.sendJSON(ctx, http.MethodPut, a.permURL+"/workflow", a.workflow); err != nil {
			return err
		}
	}
	return nil
}
