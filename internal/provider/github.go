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
	resp, err := g.do(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, RemoteRepo{}, fmt.Errorf("github: check %s/%s: %w", g.owner, spec.Name, err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		var r ghRepo
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			return false, RemoteRepo{}, err
		}
		return true, RemoteRepo(r), nil
	case http.StatusNotFound:
		return false, RemoteRepo{}, nil
	default:
		return false, RemoteRepo{}, apiError(fmt.Sprintf("github: check %s/%s", g.owner, spec.Name), resp)
	}
}

// CreateRepo creates owner/name under the authenticated user.
func (g *GitHub) CreateRepo(ctx context.Context, spec RepoSpec) (RemoteRepo, error) {
	payload, err := json.Marshal(map[string]any{
		"name": spec.Name, "description": spec.Description, "private": spec.Private,
	})
	if err != nil {
		return RemoteRepo{}, fmt.Errorf("github: create %s: %w", spec.Name, err)
	}
	url := g.baseURL + "/user/repos"
	resp, err := g.do(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return RemoteRepo{}, fmt.Errorf("github: create %s: %w", spec.Name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return RemoteRepo{}, apiError("github: create "+spec.Name, resp)
	}
	var r ghRepo
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return RemoteRepo{}, err
	}
	return RemoteRepo(r), nil
}

type ghRepo struct {
	CloneURL string `json:"clone_url"`
	HTMLURL  string `json:"html_url"`
}

// apiError builds an error from a non-success GitHub response, including the
// response body (which carries GitHub's human-readable message). An
// already-exists conflict is wrapped as ErrRepoExists so callers can detect it.
func apiError(prefix string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10)) //nolint:gosec // best-effort read of the error body for diagnostics
	msg := strings.TrimSpace(string(body))
	if resp.StatusCode == http.StatusUnprocessableEntity && strings.Contains(msg, "already exists") {
		return fmt.Errorf("%s: %w", prefix, ErrRepoExists)
	}
	if msg == "" {
		return fmt.Errorf("%s: status %d", prefix, resp.StatusCode)
	}
	return fmt.Errorf("%s: status %d: %s", prefix, resp.StatusCode, msg)
}
