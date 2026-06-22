package provider

import (
	"errors"
	"fmt"
)

// Env carries the runtime credentials/identity a provider needs.
type Env struct {
	Token   string
	Owner   string
	BaseURL string // optional; empty ⇒ the provider's default endpoint
}

// For returns a Provider for the named host. "none" yields (nil, nil).
func For(name string, env Env) (Provider, error) {
	switch name {
	case "", "none":
		return nil, nil
	case "github":
		if env.Token == "" {
			return nil, errors.New("provider github requires a token in $KEEL_GITHUB_TOKEN (or $GITHUB_TOKEN)")
		}
		if env.Owner == "" {
			return nil, errors.New("provider github requires an owner (derived from the module path, or set $KEEL_GITHUB_OWNER)")
		}
		var opts []Option
		if env.BaseURL != "" {
			opts = append(opts, WithBaseURL(env.BaseURL))
		}
		return NewGitHub(env.Token, env.Owner, opts...), nil
	default:
		return nil, fmt.Errorf("unknown provider %q", name)
	}
}
