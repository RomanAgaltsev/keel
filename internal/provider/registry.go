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

// providerSpec describes how to validate Env and build one provider.
type providerSpec struct {
	tokenErr string // returned when Env.Token is empty
	ownerErr string // returned when Env.Owner is empty
	build    func(env Env) Provider
}

// registry maps a host name to its construction recipe. Keep the per-provider
// option plumbing here so For stays a thin dispatcher.
var registry = map[string]providerSpec{
	"github": {
		tokenErr: "provider github requires a token in $KEEL_GITHUB_TOKEN (or $GITHUB_TOKEN)",
		ownerErr: "provider github requires an owner (derived from the module path, or set $KEEL_GITHUB_OWNER)",
		build: func(env Env) Provider {
			var opts []Option
			if env.BaseURL != "" {
				opts = append(opts, WithBaseURL(env.BaseURL))
			}
			return NewGitHub(env.Token, env.Owner, opts...)
		},
	},
	"gitlab": {
		tokenErr: "provider gitlab requires a token in $KEEL_GITLAB_TOKEN (or $GITLAB_TOKEN)",
		ownerErr: "provider gitlab requires an owner (derived from the module path, or set $KEEL_GITLAB_OWNER)",
		build: func(env Env) Provider {
			var opts []GitLabOption
			if env.BaseURL != "" {
				opts = append(opts, WithGitLabBaseURL(env.BaseURL))
			}
			return NewGitLab(env.Token, env.Owner, opts...)
		},
	},
	"bitbucket": {
		tokenErr: "provider bitbucket requires an access token in $KEEL_BITBUCKET_TOKEN (or $BITBUCKET_TOKEN)",
		ownerErr: "provider bitbucket requires a workspace (derived from the module path, or set $KEEL_BITBUCKET_OWNER)",
		build: func(env Env) Provider {
			var opts []BitbucketOption
			if env.BaseURL != "" {
				opts = append(opts, WithBitbucketBaseURL(env.BaseURL))
			}
			return NewBitbucket(env.Token, env.Owner, opts...)
		},
	},
	"sourcecraft": {
		tokenErr: "provider sourcecraft requires a token in $KEEL_SOURCECRAFT_TOKEN (or $SOURCECRAFT_TOKEN)",
		ownerErr: "provider sourcecraft requires an org (derived from the module path, or set $KEEL_SOURCECRAFT_OWNER)",
		build: func(env Env) Provider {
			var opts []SourceCraftOption
			if env.BaseURL != "" {
				opts = append(opts, WithSourceCraftBaseURL(env.BaseURL))
			}
			return NewSourceCraft(env.Token, env.Owner, opts...)
		},
	},
}

// For returns a Provider for the named host. "none" yields (nil, nil).
func For(name string, env Env) (Provider, error) {
	if name == "" || name == "none" {
		return nil, nil
	}
	spec, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider %q", name)
	}
	if env.Token == "" {
		return nil, errors.New(spec.tokenErr)
	}
	if env.Owner == "" {
		return nil, errors.New(spec.ownerErr)
	}
	return spec.build(env), nil
}
