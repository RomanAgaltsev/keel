package update

import (
	"fmt"
	"sort"

	"github.com/RomanAgaltsev/keel/v2/internal/lock"
	"github.com/RomanAgaltsev/keel/v2/internal/modver"
)

// State is what an update should do with one module.
type State int

const (
	// Unchanged: in the lock at the version this binary carries — skip it.
	Unchanged State = iota
	// Behind: in the lock, but this binary carries a newer version.
	Behind
	// Added: named by the recipe and absent from the lock. The repo was
	// scaffolded before the recipe gained it.
	Added
	// Orphaned: recorded in the lock but no longer named by the recipe, so its
	// files should be retracted.
	Orphaned
)

// Provenance reports where a module resolves from and at what version in this
// binary. It is the seam that keeps Resolve free of the module loader, and so
// of the filesystem and network.
type Provenance func(name string) (source, version string, err error)

// Options carries the update flags that affect module selection.
type Options struct {
	Reconfigure bool     // --reconfigure: re-render every module the recipe names
	Only        []string // --modules: restrict to these names (empty ⇒ no filter)
}

// ModuleSet is the resolved plan for every module in play.
type ModuleSet struct {
	State          map[string]State
	VersionChanged map[string]bool   // feeds Classify's v1-lock baseline heuristic
	Refreshed      map[string]string // module → version to record in the new lock
	Source         map[string]string // module → provenance ("builtin", a URL, …)
	RecipeOrder    []string          // recipe module order, for deterministic output
}

// Candidates returns the modules whose rendered files this update may write.
func (m ModuleSet) Candidates() map[string]bool {
	out := map[string]bool{}
	for name, st := range m.State {
		if st == Behind || st == Added {
			out[name] = true
		}
	}
	return out
}

// AddedModules returns the added module names in recipe order.
func (m ModuleSet) AddedModules() []string {
	var out []string
	for _, name := range m.RecipeOrder {
		if m.State[name] == Added {
			out = append(out, name)
		}
	}
	return out
}

// OrphanedModules returns the orphaned module names, sorted.
func (m ModuleSet) OrphanedModules() []string {
	var out []string
	for name, st := range m.State {
		if st == Orphaned {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// OrphanedSet is OrphanedModules as a lookup, for Classify.Input.
func (m ModuleSet) OrphanedSet() map[string]bool {
	out := map[string]bool{}
	for name, st := range m.State {
		if st == Orphaned {
			out[name] = true
		}
	}
	return out
}

// Resolve decides what happens to every module named by the lock or the recipe.
// The recipe is authoritative for composition: a module it names and the lock
// does not is Added, and a module the lock records and it does not is Orphaned.
// Callers that could not read the recipe must pass the lock's own module names,
// which yields no additions and no orphans.
func Resolve(lk lock.Lock, recipeModules []string, prov Provenance, opts Options) (ModuleSet, error) {
	ms := ModuleSet{
		State:          map[string]State{},
		VersionChanged: map[string]bool{},
		Refreshed:      map[string]string{},
		Source:         map[string]string{},
		RecipeOrder:    append([]string{}, recipeModules...),
	}
	inRecipe := setOf(recipeModules)

	locked, err := ms.resolveLocked(lk, inRecipe, prov)
	if err != nil {
		return ModuleSet{}, err
	}
	if err := ms.resolveAdded(recipeModules, locked, prov); err != nil {
		return ModuleSet{}, err
	}
	if opts.Reconfigure {
		if err := ms.promoteUnchanged(prov); err != nil {
			return ModuleSet{}, err
		}
	}
	ms.narrowTo(opts.Only)
	return ms, nil
}

// resolveLocked classifies every module the lock records, returning the set of
// names it covered so the caller can tell which recipe modules are new.
func (m ModuleSet) resolveLocked(lk lock.Lock, inRecipe map[string]bool, prov Provenance) (map[string]bool, error) {
	locked := map[string]bool{}
	for _, mod := range lk.Modules {
		locked[mod.Name] = true
		if !inRecipe[mod.Name] {
			m.State[mod.Name] = Orphaned
			continue
		}
		source, cur, err := prov(mod.Name)
		if err != nil {
			return nil, fmt.Errorf("module %q: %w", mod.Name, err)
		}
		m.Source[mod.Name] = source
		m.compareVersions(mod.Name, mod.Version, cur)
	}
	return locked, nil
}

// compareVersions sets the state for a module present in both the lock and the
// recipe, from the recorded version against the one this binary carries.
func (m ModuleSet) compareVersions(name, recorded, current string) {
	cmp, err := modver.Compare(recorded, current)
	switch {
	case err != nil:
		// Unparseable versions ⇒ treat as unchanged; never guess a bump.
		m.State[name] = Unchanged
	case cmp < 0:
		m.State[name] = Behind
		m.VersionChanged[name] = true
		m.Refreshed[name] = current
	case cmp > 0:
		// The lock is ahead of this binary (keel was downgraded). Record the
		// difference but never rewrite files backwards.
		m.State[name] = Unchanged
		m.VersionChanged[name] = true
	default:
		m.State[name] = Unchanged
	}
}

// resolveAdded marks every recipe module the lock never recorded.
func (m ModuleSet) resolveAdded(recipeModules []string, locked map[string]bool, prov Provenance) error {
	for _, name := range recipeModules {
		if locked[name] {
			continue
		}
		source, cur, err := prov(name)
		if err != nil {
			return fmt.Errorf("module %q: %w", name, err)
		}
		m.State[name] = Added
		// Added modules have no recorded baseline, so they must count as changed:
		// otherwise classifyRendered treats the current render as the baseline.
		m.VersionChanged[name] = true
		m.Refreshed[name] = cur
		m.Source[name] = source
	}
	return nil
}

// promoteUnchanged implements --reconfigure: re-render every module that is
// otherwise up to date. It never touches Orphaned, so a dropped module is not
// resurrected by asking for a re-render.
func (m ModuleSet) promoteUnchanged(prov Provenance) error {
	for name, st := range m.State {
		if st != Unchanged {
			continue
		}
		source, cur, err := prov(name)
		if err != nil {
			return fmt.Errorf("module %q: %w", name, err)
		}
		m.State[name] = Behind
		m.Refreshed[name] = cur
		m.Source[name] = source
	}
	return nil
}

// narrowTo implements --modules: everything outside the filter becomes
// Unchanged. Scoping must narrow additions and orphaning too, or `--modules
// lint-go` could still delete another module's files.
func (m ModuleSet) narrowTo(only []string) {
	if len(only) == 0 {
		return
	}
	keep := setOf(only)
	for name := range m.State {
		if !keep[name] {
			m.State[name] = Unchanged
			delete(m.Refreshed, name)
		}
	}
}

// setOf turns a name list into a lookup.
func setOf(names []string) map[string]bool {
	out := make(map[string]bool, len(names))
	for _, n := range names {
		out[n] = true
	}
	return out
}
