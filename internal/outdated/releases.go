package outdated

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrNoRelease means the repo has no "latest" release (404) — skip it.
var ErrNoRelease = errors.New("no latest release")

// ReleaseClient resolves the latest release tag for an owner/repo.
type ReleaseClient interface {
	LatestTag(ctx context.Context, repo string) (string, error)
}

// GitHubReleases queries the GitHub REST API for latest releases.
type GitHubReleases struct {
	token   string
	baseURL string
	hc      *http.Client
}

// ReleaseOption configures a GitHubReleases client.
type ReleaseOption func(*GitHubReleases)

// WithReleasesBaseURL overrides the API base URL (for tests).
func WithReleasesBaseURL(u string) ReleaseOption {
	return func(g *GitHubReleases) { g.baseURL = u }
}

// NewGitHubReleases returns a client; token may be empty (anonymous).
func NewGitHubReleases(token string, opts ...ReleaseOption) *GitHubReleases {
	g := &GitHubReleases{
		token:   token,
		baseURL: "https://api.github.com",
		hc:      &http.Client{Timeout: 30 * time.Second},
	}
	for _, o := range opts {
		o(g)
	}
	return g
}

// LatestTag returns the tag_name of repo's latest release, or ErrNoRelease on 404.
func (g *GitHubReleases) LatestTag(ctx context.Context, repo string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", g.baseURL, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if g.token != "" {
		req.Header.Set("Authorization", "Bearer "+g.token)
	}
	resp, err := g.hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("releases %s: %w", repo, err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		var r struct {
			TagName string `json:"tag_name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			return "", err
		}
		return r.TagName, nil
	case http.StatusNotFound:
		return "", ErrNoRelease
	default:
		// Include GitHub's message body (e.g. a rate-limit notice) for diagnostics.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10)) //nolint:gosec // best-effort read of the error body for diagnostics
		if msg := strings.TrimSpace(string(body)); msg != "" {
			return "", fmt.Errorf("releases %s: status %d: %s", repo, resp.StatusCode, msg)
		}
		return "", fmt.Errorf("releases %s: status %d", repo, resp.StatusCode)
	}
}
