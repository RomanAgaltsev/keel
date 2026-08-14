package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrRepoExists is returned when a create races an already-existing remote, so
// callers can distinguish it from other failures with errors.Is.
var ErrRepoExists = errors.New("remote repository already exists")

// GitHub creates and inspects repositories on GitHub via the REST API.
type GitHub struct {
	token   string
	owner   string
	baseURL string
	hc      *http.Client
}

// Option configures a GitHub provider.
type Option func(*GitHub)

// WithBaseURL overrides the API base URL (for tests).
func WithBaseURL(u string) Option { return func(g *GitHub) { g.baseURL = u } }

// NewGitHub returns a GitHub provider for owner authenticated with token.
func NewGitHub(token, owner string, opts ...Option) *GitHub {
	g := &GitHub{
		token:   token,
		owner:   owner,
		baseURL: "https://api.github.com",
		hc:      &http.Client{Timeout: 30 * time.Second},
	}
	for _, o := range opts {
		o(g)
	}
	return g
}

// Name identifies the provider.
func (g *GitHub) Name() string { return "github" }

func (g *GitHub) do(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return g.hc.Do(req)
}

// RepoExists reports whether owner/name exists.
func (g *GitHub) RepoExists(ctx context.Context, spec RepoSpec) (bool, RemoteRepo, error) {
	url := fmt.Sprintf("%s/repos/%s/%s", g.baseURL, g.owner, spec.Name)
	label := fmt.Sprintf("github: check %s/%s", g.owner, spec.Name)
	return checkRepo[ghRepo](label, func() (*http.Response, error) {
		return g.do(ctx, http.MethodGet, url, nil)
	})
}

// CreateRepo creates the repository in owner's namespace.
func (g *GitHub) CreateRepo(ctx context.Context, spec RepoSpec) (RemoteRepo, error) {
	payload, err := json.Marshal(map[string]any{
		"name": spec.Name, "description": spec.Description, "private": spec.Private,
	})
	if err != nil {
		return RemoteRepo{}, fmt.Errorf("github: create %s: %w", spec.Name, err)
	}
	resp, err := g.do(ctx, http.MethodPost, g.createURL(ctx), bytes.NewReader(payload))
	if err != nil {
		return RemoteRepo{}, fmt.Errorf("github: create %s: %w", spec.Name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return RemoteRepo{}, apiError("github: create "+spec.Name, resp, http.StatusUnprocessableEntity, "already exists")
	}
	var r ghRepo
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return RemoteRepo{}, err
	}
	return r.remote(), nil
}

// createURL returns the endpoint that creates the repository in owner's
// namespace. POST /user/repos always creates under the token's own account, so an
// owner that is an organization must go to /orgs/{owner}/repos instead. Getting
// this wrong is silent and confusing: the repo lands in the token user's
// namespace while RepoExists keeps looking at the org, so the next run's create
// fails with "already exists" for a repo the existence check just reported
// missing.
func (g *GitHub) createURL(ctx context.Context) string {
	personal := g.baseURL + "/user/repos"
	if g.owner == "" {
		return personal
	}
	login, err := g.authenticatedLogin(ctx)
	// An unavailable lookup falls back to the personal endpoint: it is the
	// pre-existing behaviour and the right answer for the common case.
	if err != nil || strings.EqualFold(login, g.owner) {
		return personal
	}
	return fmt.Sprintf("%s/orgs/%s/repos", g.baseURL, g.owner)
}

// authenticatedLogin returns the login of the user the token belongs to.
func (g *GitHub) authenticatedLogin(ctx context.Context) (string, error) {
	resp, err := g.do(ctx, http.MethodGet, g.baseURL+"/user", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github: GET /user: %s", resp.Status)
	}
	var u struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return "", err
	}
	return u.Login, nil
}

type ghRepo struct {
	CloneURL string `json:"clone_url"`
	HTMLURL  string `json:"html_url"`
}

func (r ghRepo) remote() RemoteRepo { return RemoteRepo(r) }
