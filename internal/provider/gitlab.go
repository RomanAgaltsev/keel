package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// GitLab creates and inspects projects on GitLab via the REST v4 API.
type GitLab struct {
	token   string
	owner   string
	baseURL string
	hc      *http.Client
}

// GitLabOption configures a GitLab provider.
type GitLabOption func(*GitLab)

// WithGitLabBaseURL overrides the API base URL (self-hosted instances / tests).
func WithGitLabBaseURL(u string) GitLabOption { return func(g *GitLab) { g.baseURL = u } }

// NewGitLab returns a GitLab provider for owner (namespace) authenticated with token.
func NewGitLab(token, owner string, opts ...GitLabOption) *GitLab {
	g := &GitLab{
		token:   token,
		owner:   owner,
		baseURL: "https://gitlab.com/api/v4",
		hc:      &http.Client{Timeout: 30 * time.Second},
	}
	for _, o := range opts {
		o(g)
	}
	return g
}

// Name identifies the provider.
func (g *GitLab) Name() string { return "gitlab" }

func (g *GitLab) do(ctx context.Context, method, endpoint string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+g.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return g.hc.Do(req)
}

type glProject struct {
	HTTPURLToRepo string `json:"http_url_to_repo"`
	WebURL        string `json:"web_url"`
}

func (p glProject) remote() RemoteRepo {
	return RemoteRepo{CloneURL: p.HTTPURLToRepo, HTMLURL: p.WebURL}
}

// RepoExists reports whether owner/name exists.
func (g *GitLab) RepoExists(ctx context.Context, spec RepoSpec) (bool, RemoteRepo, error) {
	encoded := url.PathEscape(g.owner + "/" + spec.Name)
	endpoint := fmt.Sprintf("%s/projects/%s", g.baseURL, encoded)
	label := fmt.Sprintf("gitlab: check %s/%s", g.owner, spec.Name)
	return checkRepo[glProject](label, func() (*http.Response, error) {
		return g.do(ctx, http.MethodGet, endpoint, nil)
	}, http.StatusBadRequest, "already been taken")
}

// CreateRepo creates name under the token user's namespace.
func (g *GitLab) CreateRepo(ctx context.Context, spec RepoSpec) (RemoteRepo, error) {
	visibility := "public"
	if spec.Private {
		visibility = "private"
	}
	payload, err := json.Marshal(map[string]any{
		"name":        spec.Name,
		"description": spec.Description,
		"visibility":  visibility,
	})
	if err != nil {
		return RemoteRepo{}, fmt.Errorf("gitlab: create %s: %w", spec.Name, err)
	}
	endpoint := g.baseURL + "/projects"
	resp, err := g.do(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return RemoteRepo{}, fmt.Errorf("gitlab: create %s: %w", spec.Name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return RemoteRepo{}, apiError("gitlab: create "+spec.Name, resp, http.StatusBadRequest, "already been taken")
	}
	var p glProject
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return RemoteRepo{}, err
	}
	return p.remote(), nil
}
