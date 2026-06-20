// Package outdated reports stale tool/action pins and stale keel modules in a
// scaffolded repository. It never modifies anything.
package outdated

import (
	"github.com/RomanAgaltsev/keel/internal/lock"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/modver"
)

// ModuleUpdate reports a keel module whose embedded version is newer than the
// version recorded in a repo's .scaffold.lock.
type ModuleUpdate struct {
	Name    string
	Current string // version in .scaffold.lock
	Latest  string // version embedded in this keel binary
}

// ModuleUpdates compares each locked builtin module against the version embedded
// in this keel binary (via l). Non-builtin modules and modules not present in the
// loader are skipped. Versions that don't parse as semver are skipped.
func ModuleUpdates(l module.Loader, locked []lock.Module) ([]ModuleUpdate, error) {
	var out []ModuleUpdate
	for _, lm := range locked {
		if lm.Source != "builtin" {
			continue
		}
		m, err := l.Load(lm.Name)
		if err != nil {
			continue // unknown / removed module
		}
		cmp, err := modver.Compare(lm.Version, m.Version)
		if err != nil {
			continue // unparseable version — skip rather than fail the whole report
		}
		if cmp < 0 {
			out = append(out, ModuleUpdate{Name: lm.Name, Current: lm.Version, Latest: m.Version})
		}
	}
	return out, nil
}
