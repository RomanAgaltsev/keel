package settings

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultPath is where keel renders and expects the settings file, relative to
// the repository root. It deliberately differs from .github/settings.yml, which
// the Probot Settings app watches and would fight over.
const DefaultPath = ".github/keel-settings.yml"

// SupportedVersion is the only schema version this build understands.
const SupportedVersion = 1

// ErrNotFound reports an absent settings file. Callers treat it as "nothing to
// do" rather than as a failure.
var ErrNotFound = errors.New("settings file not found")

// Load reads and validates the settings file at path. Unknown keys are an error:
// a typo'd key would otherwise be silently ignored, which is the most likely way
// for the file to drift away from the schema unnoticed.
func Load(path string) (Desired, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Desired{}, ErrNotFound
	}
	if err != nil {
		return Desired{}, fmt.Errorf("settings: read %q: %w", path, err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	var d Desired
	if err := dec.Decode(&d); err != nil {
		return Desired{}, fmt.Errorf("settings: parse %q: %w", path, err)
	}
	if d.Version != SupportedVersion {
		return Desired{}, fmt.Errorf("settings: %q declares version %d, this keel understands version %d", path, d.Version, SupportedVersion)
	}
	if err := d.Validate(); err != nil {
		return Desired{}, fmt.Errorf("settings: %q: %w", path, err)
	}
	return d, nil
}

// Validate rejects values the schema's types cannot express.
func (d Desired) Validate() error {
	if err := d.validateActions(); err != nil {
		return err
	}
	return d.validateRulesets()
}

// validateActions checks the two closed-vocabulary fields in the actions section.
func (d Desired) validateActions() error {
	if d.Actions == nil {
		return nil
	}
	if v := d.Actions.Allowed; v != nil {
		if !slices.Contains([]string{AllowedAll, AllowedLocalOnly, AllowedLocalAndVerified}, *v) {
			return fmt.Errorf("actions.allowed %q: want one of %q, %q, %q", *v, AllowedAll, AllowedLocalOnly, AllowedLocalAndVerified)
		}
	}
	if v := d.Actions.DefaultWorkflowPermissions; v != nil && *v != "read" && *v != "write" {
		return fmt.Errorf("actions.default_workflow_permissions %q: want \"read\" or \"write\"", *v)
	}
	return validatePatterns(d.Actions)
}

// validatePatterns checks the allow-list. It is meaningful only alongside
// local_and_verified: "all" needs no list and "local_only" admits none, so a
// pattern under either would be accepted here and silently discarded by the
// host — the kind of quiet no-op this schema exists to prevent.
func validatePatterns(a *Actions) error {
	if len(a.AllowedPatterns) == 0 {
		return nil
	}
	if a.Allowed == nil || *a.Allowed != AllowedLocalAndVerified {
		return fmt.Errorf("actions.allowed_patterns requires actions.allowed: %s", AllowedLocalAndVerified)
	}
	seen := make(map[string]bool, len(a.AllowedPatterns))
	for _, p := range a.AllowedPatterns {
		if strings.TrimSpace(p) == "" {
			return errors.New("actions.allowed_patterns: empty entry")
		}
		if seen[p] {
			return fmt.Errorf("actions.allowed_patterns: duplicate entry %q", p)
		}
		seen[p] = true
	}
	return nil
}

// validateRulesets checks the fields a ruleset cannot be reconciled without.
func (d Desired) validateRulesets() error {
	for i, rs := range d.Rulesets {
		if rs.Name == "" {
			return fmt.Errorf("rulesets[%d]: name is required (it is the identity key)", i)
		}
		if rs.Target != "" && rs.Target != "branch" {
			return fmt.Errorf("rulesets[%d] %q: target %q: only \"branch\" is supported", i, rs.Name, rs.Target)
		}
		if rs.Ref == "" {
			return fmt.Errorf("rulesets[%d] %q: ref is required", i, rs.Name)
		}
		for _, b := range rs.Bypass {
			if b != BypassRepoAdmin {
				return fmt.Errorf("rulesets[%d] %q: bypass %q: only %q is supported", i, rs.Name, b, BypassRepoAdmin)
			}
		}
	}
	return nil
}
