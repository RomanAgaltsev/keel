package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

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
	}
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
}

func (s *ghSecurityGroup) Name() string { return "security" }

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

// enabled reports whether the named analysis feature is on. An absent member
// means the feature is unavailable on this repo, which reads as off.
func (a ghAnalysisState) enabled(name string) bool {
	return a.SecurityAndAnalysis[name].Status == "enabled"
}

func (s *ghSecurityGroup) Plan(ctx context.Context, d settings.Desired) ([]settings.Change, error) {
	if d.Security == nil {
		return nil, nil
	}
	s.analysis, s.toggles = map[string]any{}, nil
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
	got := cur.enabled(name)
	if got == *want {
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
