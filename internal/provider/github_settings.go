package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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
