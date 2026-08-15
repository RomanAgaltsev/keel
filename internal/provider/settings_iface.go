package provider

import "github.com/RomanAgaltsev/keel/v2/internal/settings"

// SettingsApplier is implemented by providers that can reconcile repository
// settings. It is an optional capability, kept off the core Provider interface
// so that a provider without it stays a three-method implementation: callers
// type-assert and report the gap rather than failing.
type SettingsApplier interface {
	SettingsGroups(spec RepoSpec) []settings.Group
}
