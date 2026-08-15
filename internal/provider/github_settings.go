package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/RomanAgaltsev/keel/v2/internal/settings"
)

// SettingsGroups returns the reconcilable API surfaces for one repository.
// Each group is stateful across Plan/Apply, so the slice is built fresh per call.
func (g *GitHub) SettingsGroups(spec RepoSpec) []settings.Group {
	base := fmt.Sprintf("%s/repos/%s/%s", g.baseURL, g.owner, spec.Name)
	return []settings.Group{
		&ghRepoGroup{gh: g, url: base},
		&ghTopicsGroup{gh: g, url: base + "/topics"},
		&ghSecurityGroup{
			gh:        g,
			repoURL:   base,
			alertsURL: base + "/vulnerability-alerts",
			fixesURL:  base + "/automated-security-fixes",
		},
		&ghVulnReportingGroup{gh: g, url: base + "/private-vulnerability-reporting"},
		&ghActionsGroup{gh: g, permURL: base + "/actions/permissions"},
		&ghRulesetGroup{gh: g, url: base + "/rulesets"},
	}
}

// diffString is diffBool for string-valued fields.
func diffString(out *[]settings.Change, body map[string]any, key, field, got string, want *string) {
	if want == nil || *want == got {
		return
	}
	body[field] = *want
	*out = append(*out, settings.Change{Key: key, From: got, To: *want})
}

// readToggle reads one of GitHub's on/off repository endpoints. Verified live on
// 2026-08-15: enabled answers 204 (vulnerability-alerts, bodiless) or 200 with a
// JSON body (automated-security-fixes), and disabled answers 404 for both — so
// the status alone is authoritative and the body is deliberately ignored.
func (g *GitHub) readToggle(ctx context.Context, url string) (bool, error) {
	resp, err := g.do(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, apiError("github: get "+url, resp, 0, "")
	}
}

// writeToggle enables with PUT and disables with DELETE, the shape GitHub uses
// for its bodiless repository toggles.
func (g *GitHub) writeToggle(ctx context.Context, url string, on bool) error {
	method := http.MethodDelete
	if on {
		method = http.MethodPut
	}
	return g.sendJSON(ctx, method, url, nil)
}

// toggle pairs an endpoint with the value Plan decided to write.
type toggle struct {
	url string
	on  bool
}

// ghSecurityGroup reconciles the code-security settings. They span three
// endpoints: the security_and_analysis object on the repo, and two bodiless
// toggles.
type ghSecurityGroup struct {
	gh                           *GitHub
	repoURL, alertsURL, fixesURL string
	analysis                     map[string]any // staged security_and_analysis
	toggles                      []toggle
	unsupported                  []settings.Unsupported
}

func (s *ghSecurityGroup) Name() string { return "security" }

// Unsupported reports declared analysis keys this repository will not report, so
// they are never mistaken for drift. Populated by Plan, which is why Reconcile
// queries it afterwards.
func (s *ghSecurityGroup) Unsupported(_ settings.Desired) []settings.Unsupported {
	return s.unsupported
}

// ghAnalysisState decodes the security_and_analysis object, whose members are
// each {"status":"enabled"|"disabled"}.
//
// The whole object is absent from a private repo's response (verified
// 2026-08-15), so every member reads as disabled there — which is correct: the
// features genuinely are unavailable.
type ghAnalysisState struct {
	SecurityAndAnalysis map[string]struct {
		Status string `json:"status"`
	} `json:"security_and_analysis"`
}

// enabled reports whether the named analysis feature is on, and whether the
// provider reported it at all. An absent member reads as off, but the caller must
// know the difference: a member GitHub does not report is one keel cannot
// converge, and staging a write for it produces drift that never clears.
func (a ghAnalysisState) enabled(name string) (on, reported bool) {
	m, ok := a.SecurityAndAnalysis[name]
	return m.Status == "enabled", ok
}

func (s *ghSecurityGroup) Plan(ctx context.Context, d settings.Desired) ([]settings.Change, error) {
	if d.Security == nil {
		return nil, nil
	}
	s.analysis, s.toggles, s.unsupported = map[string]any{}, nil, nil
	var out []settings.Change
	w := d.Security

	if err := s.planAnalysis(ctx, &out, w); err != nil {
		return nil, err
	}
	if err := s.diffToggle(ctx, &out, "security.dependabot_alerts", s.alertsURL, w.DependabotAlerts); err != nil {
		return nil, err
	}
	if err := s.diffToggle(ctx, &out, "security.dependabot_security_updates", s.fixesURL, w.DependabotSecurityUpdates); err != nil {
		return nil, err
	}
	return out, nil
}

// planAnalysis reads the repo object once and diffs the three members that live
// in security_and_analysis, skipping the request entirely when none is declared.
func (s *ghSecurityGroup) planAnalysis(ctx context.Context, out *[]settings.Change, w *settings.Security) error {
	if w.DependencyGraph == nil && w.SecretScanning == nil && w.SecretScanningPushProtection == nil {
		return nil
	}
	var cur ghAnalysisState
	if err := s.gh.getJSON(ctx, s.repoURL, &cur); err != nil {
		return err
	}
	s.diffAnalysis(out, cur, "dependency_graph", w.DependencyGraph)
	s.diffAnalysis(out, cur, "secret_scanning", w.SecretScanning)
	s.diffAnalysis(out, cur, "secret_scanning_push_protection", w.SecretScanningPushProtection)
	return nil
}

// diffAnalysis stages one security_and_analysis member when it drifts.
func (s *ghSecurityGroup) diffAnalysis(out *[]settings.Change, cur ghAnalysisState, name string, want *bool) {
	if want == nil {
		return
	}
	got, reported := cur.enabled(name)
	if got == *want {
		return
	}
	if !reported {
		// GitHub omits members it will not let this repository manage — most
		// visibly dependency_graph on a public repo, which is always on and cannot
		// be turned off. Writing it is accepted and changes nothing, so treating
		// this as drift would report the same difference on every run forever.
		s.unsupported = append(s.unsupported, settings.Unsupported{
			Key:      "security." + name,
			Provider: "github",
			Reason:   "this repository does not report " + name + "; it cannot be converged here",
		})
		return
	}
	status := "disabled"
	if *want {
		status = "enabled"
	}
	s.analysis[name] = map[string]any{"status": status}
	*out = append(*out, settings.Change{
		Key:  "security." + name,
		From: strconv.FormatBool(got),
		To:   strconv.FormatBool(*want),
	})
}

// diffToggle reads a bodiless toggle and stages a write when it drifts.
func (s *ghSecurityGroup) diffToggle(ctx context.Context, out *[]settings.Change, key, url string, want *bool) error {
	if want == nil {
		return nil
	}
	got, err := s.gh.readToggle(ctx, url)
	if err != nil {
		return err
	}
	if got == *want {
		return nil
	}
	s.toggles = append(s.toggles, toggle{url: url, on: *want})
	*out = append(*out, settings.Change{Key: key, From: strconv.FormatBool(got), To: strconv.FormatBool(*want)})
	return nil
}

func (s *ghSecurityGroup) Apply(ctx context.Context, _ []settings.Change) error {
	if len(s.analysis) > 0 {
		body := map[string]any{"security_and_analysis": s.analysis}
		if err := s.gh.sendJSON(ctx, http.MethodPatch, s.repoURL, body); err != nil {
			return err
		}
	}
	for _, t := range s.toggles {
		if err := s.gh.writeToggle(ctx, t.url, t.on); err != nil {
			return err
		}
	}
	return nil
}

// ghVulnReportingGroup reconciles private vulnerability reporting, a single
// bodiless toggle kept separate because it is neither part of the repo object
// nor of the security_and_analysis block.
type ghVulnReportingGroup struct {
	gh   *GitHub
	url  string
	want *bool
}

func (v *ghVulnReportingGroup) Name() string { return "vuln-reporting" }

func (v *ghVulnReportingGroup) Plan(ctx context.Context, d settings.Desired) ([]settings.Change, error) {
	v.want = nil
	if d.Security == nil || d.Security.PrivateVulnerabilityReporting == nil {
		return nil, nil
	}
	want := *d.Security.PrivateVulnerabilityReporting
	got, err := v.gh.readToggle(ctx, v.url)
	if err != nil {
		return nil, err
	}
	if got == want {
		return nil, nil
	}
	v.want = &want
	return []settings.Change{{
		Key:  "security.private_vulnerability_reporting",
		From: strconv.FormatBool(got),
		To:   strconv.FormatBool(want),
	}}, nil
}

func (v *ghVulnReportingGroup) Apply(ctx context.Context, _ []settings.Change) error {
	if v.want == nil {
		return nil
	}
	return v.gh.writeToggle(ctx, v.url, *v.want)
}

// getJSON issues a GET and decodes a JSON body into out.
func (g *GitHub) getJSON(ctx context.Context, url string, out any) error {
	resp, err := g.do(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return apiError("github: get "+url, resp, 0, "")
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// sendJSON issues a write with an optional JSON body and accepts any 2xx.
// A nil body sends no body at all, which several GitHub toggles require.
func (g *GitHub) sendJSON(ctx context.Context, method, url string, body any) error {
	var rdr *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	var resp *http.Response
	var err error
	if rdr == nil {
		resp, err = g.do(ctx, method, url, nil)
	} else {
		resp, err = g.do(ctx, method, url, rdr)
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return apiError("github: "+method+" "+url, resp, 0, "")
	}
	return nil
}

// diffBool stages field into body and records a Change when want is declared and
// differs from got. Undeclared (nil) means "leave alone" and is the whole reason
// the schema uses pointers.
func diffBool(out *[]settings.Change, body map[string]any, key, field string, got bool, want *bool) {
	if want == nil || *want == got {
		return
	}
	body[field] = *want
	*out = append(*out, settings.Change{Key: key, From: strconv.FormatBool(got), To: strconv.FormatBool(*want)})
}

// ghRepoGroup reconciles the general repository options, all of which the single
// PATCH /repos/{owner}/{repo} call sets at once.
type ghRepoGroup struct {
	gh   *GitHub
	url  string
	body map[string]any // staged by Plan, sent by Apply
}

func (r *ghRepoGroup) Name() string { return "repository" }

// ghRepoState is the subset of the repo object this group reconciles.
type ghRepoState struct {
	AllowSquashMerge    bool `json:"allow_squash_merge"`
	AllowMergeCommit    bool `json:"allow_merge_commit"`
	AllowRebaseMerge    bool `json:"allow_rebase_merge"`
	AllowAutoMerge      bool `json:"allow_auto_merge"`
	DeleteBranchOnMerge bool `json:"delete_branch_on_merge"`
	HasWiki             bool `json:"has_wiki"`
	HasProjects         bool `json:"has_projects"`
}

func (r *ghRepoGroup) Plan(ctx context.Context, d settings.Desired) ([]settings.Change, error) {
	if d.Repository == nil {
		return nil, nil
	}
	var cur ghRepoState
	if err := r.gh.getJSON(ctx, r.url, &cur); err != nil {
		return nil, err
	}
	body := map[string]any{}
	var out []settings.Change
	w := d.Repository
	diffBool(&out, body, "repository.allow_squash_merge", "allow_squash_merge", cur.AllowSquashMerge, w.AllowSquashMerge)
	diffBool(&out, body, "repository.allow_merge_commit", "allow_merge_commit", cur.AllowMergeCommit, w.AllowMergeCommit)
	diffBool(&out, body, "repository.allow_rebase_merge", "allow_rebase_merge", cur.AllowRebaseMerge, w.AllowRebaseMerge)
	diffBool(&out, body, "repository.allow_auto_merge", "allow_auto_merge", cur.AllowAutoMerge, w.AllowAutoMerge)
	diffBool(&out, body, "repository.delete_branch_on_merge", "delete_branch_on_merge", cur.DeleteBranchOnMerge, w.DeleteBranchOnMerge)
	diffBool(&out, body, "repository.has_wiki", "has_wiki", cur.HasWiki, w.HasWiki)
	diffBool(&out, body, "repository.has_projects", "has_projects", cur.HasProjects, w.HasProjects)
	r.body = body
	return out, nil
}

func (r *ghRepoGroup) Apply(ctx context.Context, _ []settings.Change) error {
	if len(r.body) == 0 {
		return nil
	}
	return r.gh.sendJSON(ctx, http.MethodPatch, r.url, r.body)
}

// ghTopicsGroup reconciles repository topics. PUT replaces the whole list, so
// declaring topics hands keel ownership of all of them.
type ghTopicsGroup struct {
	gh   *GitHub
	url  string
	want []string
}

func (t *ghTopicsGroup) Name() string { return "topics" }

func (t *ghTopicsGroup) Plan(ctx context.Context, d settings.Desired) ([]settings.Change, error) {
	if d.Repository == nil || d.Repository.Topics == nil {
		return nil, nil
	}
	var cur struct {
		Names []string `json:"names"`
	}
	if err := t.gh.getJSON(ctx, t.url, &cur); err != nil {
		return nil, err
	}
	want := *d.Repository.Topics
	if equalStrings(cur.Names, want) {
		return nil, nil
	}
	t.want = want
	return []settings.Change{{
		Key:  "repository.topics",
		From: fmt.Sprint(cur.Names),
		To:   fmt.Sprint(want),
	}}, nil
}

func (t *ghTopicsGroup) Apply(ctx context.Context, _ []settings.Change) error {
	if t.want == nil {
		return nil
	}
	return t.gh.sendJSON(ctx, http.MethodPut, t.url, map[string]any{"names": t.want})
}

// equalStrings compares two lists order-sensitively. Topic order is meaningful
// to the API's round-trip, so a reorder is a real change.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
	if err := a.planAllowed(ctx, &out, d.Actions.Allowed); err != nil {
		return nil, err
	}
	if err := a.planWorkflow(ctx, &out, d.Actions); err != nil {
		return nil, err
	}
	return out, nil
}

// planAllowed reads the effective policy — which needs the selected-actions
// endpoint too whenever allowed_actions is "selected" — and stages the writes.
//
// The selected-actions read is deliberately guarded: verified live on
// 2026-08-15, that endpoint answers 409 Conflict (not 404, not defaults) unless
// allowed_actions is already "selected", so reading it unconditionally would
// turn every ordinary repo into a group failure.
func (a *ghActionsGroup) planAllowed(ctx context.Context, out *[]settings.Change, want *string) error {
	if want == nil {
		return nil
	}
	got, err := a.readAllowed(ctx)
	if err != nil {
		return err
	}
	if got == *want {
		return nil
	}
	// "enabled" is required by the endpoint — omitting it is a 422, caught by the
	// 2026-08-15 acceptance run — and the current value is echoed back rather than
	// assumed, so setting a policy never switches Actions on or off as a side effect.
	switch *want {
	case settings.AllowedLocalAndVerified:
		a.perm = map[string]any{"enabled": a.enabled, "allowed_actions": "selected"}
		a.selected = map[string]any{"github_owned_allowed": true, "verified_allowed": true, "patterns_allowed": []string{}}
	default:
		a.perm = map[string]any{"enabled": a.enabled, "allowed_actions": *want}
	}
	*out = append(*out, settings.Change{Key: "actions.allowed", From: got, To: *want})
	return nil
}

// readAllowed returns the effective policy in keel's vocabulary, resolving
// "selected" into local_and_verified when the selected-actions payload says so.
func (a *ghActionsGroup) readAllowed(ctx context.Context) (string, error) {
	var cur struct {
		Enabled        bool   `json:"enabled"`
		AllowedActions string `json:"allowed_actions"`
	}
	if err := a.gh.getJSON(ctx, a.permURL, &cur); err != nil {
		return "", err
	}
	a.enabled = cur.Enabled
	if cur.AllowedActions != "selected" {
		return cur.AllowedActions, nil
	}
	var sel struct {
		GitHubOwnedAllowed bool `json:"github_owned_allowed"`
		VerifiedAllowed    bool `json:"verified_allowed"`
	}
	if err := a.gh.getJSON(ctx, a.permURL+"/selected-actions", &sel); err != nil {
		return "", err
	}
	if sel.GitHubOwnedAllowed && sel.VerifiedAllowed {
		return settings.AllowedLocalAndVerified, nil
	}
	return cur.AllowedActions, nil
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

func (a *ghActionsGroup) Apply(ctx context.Context, _ []settings.Change) error {
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

func (r *ghRulesetGroup) Apply(ctx context.Context, _ []settings.Change) error {
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
