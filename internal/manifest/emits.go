package manifest

import (
	"fmt"
	"slices"
	"strings"
)

// Needs vocabulary. It is closed: a repository capability a module can ask for
// has to be something repo-settings knows how to grant, and an unrecognised
// entry would be silently dropped -- the exact failure this contract removes.
const (
	// NeedCanApprovePullRequestReviews is required by any module whose workflows
	// open a pull request with the default GITHUB_TOKEN. release-please does.
	NeedCanApprovePullRequestReviews = "can_approve_pull_request_reviews"
)

// knownNeeds is every legal Emits.Needs entry.
var knownNeeds = []string{NeedCanApprovePullRequestReviews}

// Emits is what a module contributes to the repository's CI contract: the facts
// a sibling module -- in practice repo-settings-* -- would otherwise have to
// know about this one by hand, and get wrong when this one changes.
//
// An absent block means the module contributes nothing, which is the right
// default for a module that emits no workflows.
type Emits struct {
	// Checks are the status-check contexts this module's workflows report, named
	// exactly as the host names them. A matrixed job reports one context per
	// matrix cell, so a module that wants a single stable context emits an
	// aggregate gate job and names that.
	Checks []string `yaml:"checks,omitempty"`

	// Actions are the third-party actions its workflows use, without a version:
	// the allow-list is rendered as `owner/repo@*`, so pinning here would be
	// duplicated precision that drifts against the workflow's own pin.
	Actions []string `yaml:"actions,omitempty"`

	// Needs are repository capabilities its workflows require in order to
	// function. Entries must come from knownNeeds.
	Needs []string `yaml:"needs,omitempty"`
}

// Validate reports whether the block is well formed.
func (e Emits) Validate() error {
	if err := validateList("emits.checks", e.Checks); err != nil {
		return err
	}
	if err := validateList("emits.actions", e.Actions); err != nil {
		return err
	}
	for _, a := range e.Actions {
		if strings.Contains(a, "@") {
			return fmt.Errorf("emits.actions %q: name the action without a version; the allow-list is rendered as owner/repo@*", a)
		}
	}
	if err := validateList("emits.needs", e.Needs); err != nil {
		return err
	}
	for _, n := range e.Needs {
		if !slices.Contains(knownNeeds, n) {
			return fmt.Errorf("emits.needs %q: unknown capability, want one of %v", n, knownNeeds)
		}
	}
	return nil
}

// validateList rejects empty and duplicated entries in one of the three lists.
func validateList(field string, vals []string) error {
	seen := make(map[string]bool, len(vals))
	for _, v := range vals {
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("%s: empty entry", field)
		}
		if seen[v] {
			return fmt.Errorf("%s: duplicate entry %q", field, v)
		}
		seen[v] = true
	}
	return nil
}
