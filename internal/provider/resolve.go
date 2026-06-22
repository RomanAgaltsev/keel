package provider

import (
	"fmt"
	"os"
	"strings"
)

// EnvSpec declares how a provider reads its credentials/host from the environment.
type EnvSpec struct {
	TokenVars []string // token env vars in priority order
	OwnerVar  string   // owner override env var
	URLVar    string   // base-URL override env var
}

// envSpecs is the single source of per-provider environment knowledge. Providers
// added in later plans register their EnvSpec here alongside their For() case.
var envSpecs = map[string]EnvSpec{
	"github": {
		TokenVars: []string{"KEEL_GITHUB_TOKEN", "GITHUB_TOKEN"},
		OwnerVar:  "KEEL_GITHUB_OWNER",
		URLVar:    "KEEL_GITHUB_URL",
	},
	"gitlab": {
		TokenVars: []string{"KEEL_GITLAB_TOKEN", "GITLAB_TOKEN"},
		OwnerVar:  "KEEL_GITLAB_OWNER",
		URLVar:    "KEEL_GITLAB_URL",
	},
}

// Resolve constructs a provider by name, reading its token / owner / base-URL from
// the environment. The owner defaults to the module path's second segment
// (github.com/Owner/repo → Owner), overridable by the provider's OwnerVar. It
// returns (nil, nil) for "none"/"".
func Resolve(name, modulePath string) (Provider, error) {
	if name == "" || name == "none" {
		return For(name, Env{})
	}
	spec, ok := envSpecs[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider %q", name)
	}
	env := Env{
		Token:   firstEnv(spec.TokenVars...),
		Owner:   ownerFromModulePath(modulePath),
		BaseURL: os.Getenv(spec.URLVar),
	}
	if v := os.Getenv(spec.OwnerVar); v != "" {
		env.Owner = v
	}
	return For(name, env)
}

// firstEnv returns the first non-empty environment variable among keys.
func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

// ownerFromModulePath extracts the owner from a module path like
// "github.com/Owner/repo" → "Owner". Empty if it can't be determined.
func ownerFromModulePath(mp string) string {
	parts := strings.Split(mp, "/")
	if len(parts) >= 3 {
		return parts[1]
	}
	return ""
}
