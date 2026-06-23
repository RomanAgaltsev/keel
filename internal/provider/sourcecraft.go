package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SourceCraft creates and inspects repositories on SourceCraft via its REST API.
type SourceCraft struct {
	token   string
	owner   string // org slug
	baseURL string
	hc      *http.Client
}

// SourceCraftOption configures a SourceCraft provider.
type SourceCraftOption func(*SourceCraft)

// WithSourceCraftBaseURL overrides the API base URL (for tests).
func WithSourceCraftBaseURL(u string) SourceCraftOption {
	return func(s *SourceCraft) { s.baseURL = u }
}

// NewSourceCraft returns a SourceCraft provider for owner (org) authed with a PAT.
func NewSourceCraft(token, owner string, opts ...SourceCraftOption) *SourceCraft {
	s := &SourceCraft{
		token:   token,
		owner:   owner,
		baseURL: "https://api.sourcecraft.tech",
		hc:      &http.Client{Timeout: 30 * time.Second},
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Name identifies the provider.
func (s *SourceCraft) Name() string { return "sourcecraft" }

func (s *SourceCraft) do(ctx context.Context, method, endpoint string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return s.hc.Do(req)
}

// scCloneURL is the nested clone_url object (Task 1: #/definitions/CloneURL).
type scCloneURL struct {
	HTTPS string `json:"https"`
	SSH   string `json:"ssh"`
}

// scRepo mirrors the SourceCraft repository response (Task 1: clone_url is a
// nested {https, ssh} object; web_url is a plain string).
type scRepo struct {
	CloneURL scCloneURL `json:"clone_url"`
	WebURL   string     `json:"web_url"`
}

func (r scRepo) remote() RemoteRepo {
	return RemoteRepo{CloneURL: r.CloneURL.HTTPS, HTMLURL: r.WebURL}
}

// RepoExists reports whether org/name exists.
func (s *SourceCraft) RepoExists(ctx context.Context, spec RepoSpec) (bool, RemoteRepo, error) {
	// Task 1: single-repo GET path is /repos/{org_slug}/{repo_slug} (no /orgs prefix).
	endpoint := fmt.Sprintf("%s/repos/%s/%s", s.baseURL, s.owner, spec.Name)
	label := fmt.Sprintf("sourcecraft: check %s/%s", s.owner, spec.Name)
	return checkRepo[scRepo](label, func() (*http.Response, error) {
		return s.do(ctx, http.MethodGet, endpoint, nil)
	})
}

// CreateRepo creates org/name.
func (s *SourceCraft) CreateRepo(ctx context.Context, spec RepoSpec) (RemoteRepo, error) {
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
		return RemoteRepo{}, fmt.Errorf("sourcecraft: create %s: %w", spec.Name, err)
	}
	endpoint := fmt.Sprintf("%s/orgs/%s/repos", s.baseURL, s.owner)
	resp, err := s.do(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return RemoteRepo{}, fmt.Errorf("sourcecraft: create %s: %w", spec.Name, err)
	}
	defer resp.Body.Close()
	// Task 1: create returns 201 (no documented 200).
	if resp.StatusCode != http.StatusCreated {
		return RemoteRepo{}, apiError("sourcecraft: create "+spec.Name, resp, http.StatusConflict, "already exists") // Task 1: conflict status/body undocumented; verify live (Task 4)
	}
	var r scRepo
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return RemoteRepo{}, err
	}
	return r.remote(), nil
}
