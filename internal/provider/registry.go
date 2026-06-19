package provider

import (
	"errors"
	"fmt"
)

// Env carries the runtime credentials/identity a provider needs.
type Env struct {
	Token string
	Owner string
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
		return NewGitHub(env.Token, env.Owner), nil
	default:
		return nil, fmt.Errorf("unknown provider %q", name)
	}
}
