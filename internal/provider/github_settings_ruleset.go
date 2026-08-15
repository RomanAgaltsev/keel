package provider

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/RomanAgaltsev/keel/v2/internal/settings"
)

// repoAdminRoleID is GitHub's RepositoryRole id for "admin", used as the bypass
// actor. Verified live 2026-08-15 (spec §11 item 7): the created ruleset reads
// back "current_user_can_bypass":"always", and a direct push to a protected
// branch succeeded. A wrong id yields a ruleset whose bypass silently does
// nothing, discovered only the first time a push is refused.
const repoAdminRoleID = 5

// planGatedMessage is the substring GitHub returns when rulesets are unavailable
// on the repo's plan. Verified live 2026-08-15: POST /rulesets on a private repo
// on the Free plan answers 403 with this text, while the identical payload
// succeeds once the repo is public.
const planGatedMessage = "Upgrade to GitHub Pro"

// ghRulesetGroup reconciles branch rulesets, matched by name. A ruleset created
// by hand under a different name is invisible to keel and stays untouched.
//
// Unlike the per-key groups this diffs whole rulesets: GitHub returns rules as an
// unordered array of typed objects, and a key-by-key normalizer would carry more
// risk than the coarse comparison it replaces.
type ghRulesetGroup struct {
	gh          *GitHub
	url         string
	pending     []pendingRuleset
	unsupported []settings.Unsupported
}

// pendingRuleset is one ruleset write staged by Plan. id == 0 means create.
type pendingRuleset struct {
	id   int
	body map[string]any
}

// ghRulesetRef is the id/name pair the list endpoint returns.
type ghRulesetRef struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func (r *ghRulesetGroup) Name() string { return "ruleset" }

// Unsupported reports rulesets this repository's plan cannot accept. It is
// populated by Plan, because the limit is only visible in the API's answer —
// which is why Reconcile queries this after Plan rather than before.
func (r *ghRulesetGroup) Unsupported(_ settings.Desired) []settings.Unsupported {
	return r.unsupported
}

func (r *ghRulesetGroup) Plan(ctx context.Context, d settings.Desired) ([]settings.Change, error) {
	r.pending, r.unsupported = nil, nil
	if len(d.Rulesets) == 0 {
		return nil, nil
	}
	var existing []ghRulesetRef
	if err := r.gh.getJSON(ctx, r.url, &existing); err != nil {
		if r.notePlanGated(d, err) {
			return nil, nil
		}
		return nil, err
	}
	var out []settings.Change
	for _, want := range d.Rulesets {
		change, err := r.planOne(ctx, want, existing)
		if err != nil {
			return nil, err
		}
		if change != nil {
			out = append(out, *change)
		}
	}
	return out, nil
}

// planOne stages the create or update for a single declared ruleset, returning
// nil when it already matches.
func (r *ghRulesetGroup) planOne(ctx context.Context, want settings.Ruleset, existing []ghRulesetRef) (*settings.Change, error) {
	body := rulesetBody(want)
	id := 0
	for _, e := range existing {
		if e.Name == want.Name {
			id = e.ID
			break
		}
	}
	if id == 0 {
		r.pending = append(r.pending, pendingRuleset{body: body})
		return &settings.Change{
			Key:  fmt.Sprintf("ruleset.%q", want.Name),
			From: "absent",
			To:   rulesetSummary(body),
		}, nil
	}
	var cur map[string]any
	if err := r.gh.getJSON(ctx, fmt.Sprintf("%s/%d", r.url, id), &cur); err != nil {
		return nil, err
	}
	if rulesetSummary(cur) == rulesetSummary(body) {
		return nil, nil
	}
	r.pending = append(r.pending, pendingRuleset{id: id, body: body})
	return &settings.Change{
		Key:  fmt.Sprintf("ruleset.%q", want.Name),
		From: rulesetSummary(cur),
		To:   rulesetSummary(body),
	}, nil
}

// notePlanGated converts a plan-gating 403 into Unsupported entries and reports
// whether it did. This is not a failure the user can fix by correcting a token
// or a settings file, so it must not be reported as one.
func (r *ghRulesetGroup) notePlanGated(d settings.Desired, err error) bool {
	if !strings.Contains(err.Error(), planGatedMessage) {
		return false
	}
	for _, rs := range d.Rulesets {
		r.unsupported = append(r.unsupported, settings.Unsupported{
			Key:      fmt.Sprintf("ruleset.%q", rs.Name),
			Provider: "github",
			Reason:   "rulesets need GitHub Pro on a private repository (" + planGatedMessage + ")",
		})
	}
	return true
}

func (r *ghRulesetGroup) Apply(ctx context.Context) error {
	for _, p := range r.pending {
		if p.id == 0 {
			if err := r.gh.sendJSON(ctx, http.MethodPost, r.url, p.body); err != nil {
				return err
			}
			continue
		}
		if err := r.gh.sendJSON(ctx, http.MethodPut, fmt.Sprintf("%s/%d", r.url, p.id), p.body); err != nil {
			return err
		}
	}
	return nil
}

// rulesetBody renders keel's ruleset vocabulary into GitHub's array-of-rules
// shape. block_force_push and block_deletion are separate rule objects
// (non_fast_forward, deletion), not booleans on the ruleset.
func rulesetBody(rs settings.Ruleset) map[string]any {
	rules := []map[string]any{}
	if rs.RequiredApprovingReviews != nil {
		rules = append(rules, map[string]any{
			"type": "pull_request",
			"parameters": map[string]any{
				"required_approving_review_count":   *rs.RequiredApprovingReviews,
				"dismiss_stale_reviews_on_push":     false,
				"require_code_owner_review":         false,
				"require_last_push_approval":        false,
				"required_review_thread_resolution": false,
			},
		})
	}
	if len(rs.RequiredStatusChecks) > 0 {
		checks := make([]map[string]any, 0, len(rs.RequiredStatusChecks))
		for _, c := range rs.RequiredStatusChecks {
			checks = append(checks, map[string]any{"context": c})
		}
		rules = append(rules, map[string]any{
			"type": "required_status_checks",
			"parameters": map[string]any{
				"strict_required_status_checks_policy": false,
				"required_status_checks":               checks,
			},
		})
	}
	if rs.RequiredLinearHistory != nil && *rs.RequiredLinearHistory {
		rules = append(rules, map[string]any{"type": "required_linear_history"})
	}
	if rs.BlockForcePush != nil && *rs.BlockForcePush {
		rules = append(rules, map[string]any{"type": "non_fast_forward"})
	}
	if rs.BlockDeletion != nil && *rs.BlockDeletion {
		rules = append(rules, map[string]any{"type": "deletion"})
	}

	bypass := []map[string]any{}
	for _, b := range rs.Bypass {
		if b == settings.BypassRepoAdmin {
			bypass = append(bypass, map[string]any{
				"actor_id": repoAdminRoleID, "actor_type": "RepositoryRole", "bypass_mode": "always",
			})
		}
	}

	return map[string]any{
		"name":        rs.Name,
		"target":      "branch",
		"enforcement": "active",
		"conditions": map[string]any{
			"ref_name": map[string]any{"include": []string{"refs/heads/" + rs.Ref}, "exclude": []string{}},
		},
		"bypass_actors": bypass,
		"rules":         rules,
	}
}

// rulesetSummary canonicalizes a ruleset — ours or GitHub's — into a stable
// string used both for comparison and for the change's From/To. Sorting makes it
// insensitive to the order GitHub happens to return rules in.
func rulesetSummary(m map[string]any) string {
	var rules []any
	switch got := m["rules"].(type) {
	case []any:
		rules = got
	case []map[string]any:
		for _, r := range got {
			rules = append(rules, any(r))
		}
	}
	parts := make([]string, 0, len(rules))
	for _, raw := range rules {
		r, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		parts = append(parts, summarizeRule(r))
	}
	sort.Strings(parts)
	return strings.Join(parts, " ")
}

// summarizeRule renders one rule object into its canonical fragment.
func summarizeRule(r map[string]any) string {
	typ, _ := r["type"].(string)
	p, _ := r["parameters"].(map[string]any)
	switch typ {
	case "pull_request":
		return fmt.Sprintf("pull_request(reviews=%v)", numberOf(p["required_approving_review_count"]))
	case "required_status_checks":
		return "checks(" + contextsOf(p["required_status_checks"]) + ")"
	default:
		return typ
	}
}

// numberOf normalizes JSON numbers (float64) and Go ints to a comparable value.
func numberOf(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}

// contextsOf extracts and sorts the check contexts from either representation.
func contextsOf(v any) string {
	var names []string
	switch list := v.(type) {
	case []any:
		for _, c := range list {
			if m, ok := c.(map[string]any); ok {
				if s, ok := m["context"].(string); ok {
					names = append(names, s)
				}
			}
		}
	case []map[string]any:
		for _, m := range list {
			if s, ok := m["context"].(string); ok {
				names = append(names, s)
			}
		}
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}
