package provider

import (
	"context"
	"net/http"
	"strconv"

	"github.com/RomanAgaltsev/keel/v2/internal/settings"
)

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

func (s *ghSecurityGroup) Apply(ctx context.Context) error {
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

func (v *ghVulnReportingGroup) Apply(ctx context.Context) error {
	if v.want == nil {
		return nil
	}
	return v.gh.writeToggle(ctx, v.url, *v.want)
}
