// Package godeps resolves Go module versions against the module proxy. It is the
// dependency counterpart to internal/outdated's GitHub release lookup, which
// cannot serve module paths: golang.org/x/tools has no owner/repo pair, so
// outdated's Pin.Repo() would yield "golang.org/x".
package godeps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Resolver reports the latest version of a Go module path.
type Resolver interface {
	LatestVersion(ctx context.Context, modPath string) (string, error)
}

// Proxy queries a Go module proxy.
type Proxy struct {
	baseURL string
	hc      *http.Client
}

// ProxyOption configures a Proxy.
type ProxyOption func(*Proxy)

// WithBaseURL overrides the proxy base URL (for tests).
func WithBaseURL(u string) ProxyOption {
	return func(p *Proxy) { p.baseURL = strings.TrimSuffix(u, "/") }
}

// NewProxy returns a client against proxy.golang.org unless overridden.
func NewProxy(opts ...ProxyOption) *Proxy {
	p := &Proxy{
		baseURL: "https://proxy.golang.org",
		hc:      &http.Client{Timeout: 30 * time.Second},
	}
	for _, o := range opts {
		o(p)
	}
	return p
}

// EscapePath applies the module proxy's case encoding: every capital letter
// becomes "!" followed by its lowercase form, so that case-insensitive
// filesystems cannot collide two distinct module paths. An unescaped capital
// yields a 404, which would look exactly like "no such module".
func EscapePath(p string) string {
	var b strings.Builder
	for _, r := range p {
		if r >= 'A' && r <= 'Z' {
			b.WriteByte('!')
			b.WriteRune(r + ('a' - 'A'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// LatestVersion returns the proxy's @latest version for modPath.
func (p *Proxy) LatestVersion(ctx context.Context, modPath string) (string, error) {
	url := fmt.Sprintf("%s/%s/@latest", p.baseURL, EscapePath(modPath))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := p.hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", modPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("resolve %q: proxy returned %s", modPath, resp.Status)
	}
	var body struct {
		Version string `json:"Version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("resolve %q: %w", modPath, err)
	}
	if body.Version == "" {
		return "", errors.New("resolve " + modPath + ": proxy returned no version")
	}
	return body.Version, nil
}
