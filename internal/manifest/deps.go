package manifest

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Dep is one third-party module that a template module's emitted code imports
// directly. Modules declare only their direct imports; `go mod tidy`, which the
// scaffolded repo runs as the first step of its own `task ci`, computes the
// indirect closure and writes go.sum.
type Dep struct {
	Path    string `yaml:"path"`
	Version string `yaml:"version"`
}

// goSemverRE matches a Go module version: a leading v, three numeric parts, and
// an optional pre-release or build suffix.
var goSemverRE = regexp.MustCompile(`^v\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$`)

// ValidateDeps reports whether a module's deps block is well formed. language is
// the declaring module's own language: only Go modules may declare dependencies
// today, because no Rust consumer exists to design the cargo path against.
func ValidateDeps(deps []Dep, language string) error {
	if len(deps) == 0 {
		return nil
	}
	if language != "go" {
		return fmt.Errorf("deps: only go modules can declare dependencies; this module is %q and cargo support is unbuilt", language)
	}
	seen := make(map[string]bool, len(deps))
	for _, d := range deps {
		if err := validateDep(d, seen); err != nil {
			return err
		}
		seen[d.Path] = true
	}
	return nil
}

// validateDep checks one entry against the rules in the design's §2.3.
func validateDep(d Dep, seen map[string]bool) error {
	switch {
	case strings.TrimSpace(d.Path) == "":
		return errors.New("deps: empty path")
	case strings.Contains(d.Path, "@"):
		return fmt.Errorf("deps %q: put the version in the version field, not in the path", d.Path)
	case strings.TrimSpace(d.Version) == "":
		return fmt.Errorf("deps %q: empty version", d.Path)
	case !goSemverRE.MatchString(d.Version):
		return fmt.Errorf("deps %q: version %q must be v-prefixed semver, e.g. v0.38.0", d.Path, d.Version)
	case seen[d.Path]:
		return fmt.Errorf("deps: duplicate path %q", d.Path)
	}
	return nil
}
