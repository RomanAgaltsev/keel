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

// Bitbucket creates and inspects repositories on Bitbucket Cloud via the REST 2.0 API.
type Bitbucket struct {
	token   string
	owner   string // workspace
	baseURL string
	hc      *http.Client
}

// BitbucketOption configures a Bitbucket provider.
type BitbucketOption func(*Bitbucket)

// WithBitbucketBaseURL overrides the API base URL (for tests).
func WithBitbucketBaseURL(u string) BitbucketOption { return func(b *Bitbucket) { b.baseURL = u } }

// NewBitbucket returns a Bitbucket provider for owner (workspace) authed with token.
func NewBitbucket(token, owner string, opts ...BitbucketOption) *Bitbucket {
	b := &Bitbucket{
		token:   token,
		owner:   owner,
		baseURL: "https://api.bitbucket.org/2.0",
		hc:      &http.Client{Timeout: 30 * time.Second},
	}
	for _, o := range opts {
		o(b)
	}
	return b
}

// Name identifies the provider.
func (b *Bitbucket) Name() string { return "bitbucket" }

func (b *Bitbucket) do(ctx context.Context, method, endpoint string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+b.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return b.hc.Do(req)
}

type bbRepo struct {
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
		Clone []struct {
			Name string `json:"name"`
			Href string `json:"href"`
		} `json:"clone"`
	} `json:"links"`
}

func (r bbRepo) remote() RemoteRepo {
	var clone string
	for _, c := range r.Links.Clone {
		if c.Name == "https" {
			clone = c.Href
			break
		}
	}
	return RemoteRepo{CloneURL: clone, HTMLURL: r.Links.HTML.Href}
}

// RepoExists reports whether workspace/name exists.
func (b *Bitbucket) RepoExists(ctx context.Context, spec RepoSpec) (bool, RemoteRepo, error) {
	endpoint := fmt.Sprintf("%s/repositories/%s/%s", b.baseURL, b.owner, spec.Name)
	resp, err := b.do(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, RemoteRepo{}, fmt.Errorf("bitbucket: check %s/%s: %w", b.owner, spec.Name, err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		var r bbRepo
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			return false, RemoteRepo{}, err
		}
		return true, r.remote(), nil
	case http.StatusNotFound:
		return false, RemoteRepo{}, nil
	default:
		return false, RemoteRepo{}, apiError(fmt.Sprintf("bitbucket: check %s/%s", b.owner, spec.Name), resp, http.StatusBadRequest, "already exists")
	}
}

// CreateRepo creates workspace/name.
func (b *Bitbucket) CreateRepo(ctx context.Context, spec RepoSpec) (RemoteRepo, error) {
	payload, err := json.Marshal(map[string]any{
		"scm":         "git",
		"is_private":  spec.Private,
		"description": spec.Description,
	})
	if err != nil {
		return RemoteRepo{}, fmt.Errorf("bitbucket: create %s: %w", spec.Name, err)
	}
	endpoint := fmt.Sprintf("%s/repositories/%s/%s", b.baseURL, b.owner, spec.Name)
	resp, err := b.do(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return RemoteRepo{}, fmt.Errorf("bitbucket: create %s: %w", spec.Name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return RemoteRepo{}, apiError("bitbucket: create "+spec.Name, resp, http.StatusBadRequest, "already exists")
	}
	var r bbRepo
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return RemoteRepo{}, err
	}
	return r.remote(), nil
}
