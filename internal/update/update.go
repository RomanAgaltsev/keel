// Package update computes, from a repo's .scaffold.lock and the current module
// renders, how each file should change when re-applying evolved templates. It is
// pure (no filesystem, no git, no network); callers supply the rendered content
// and a hash-of-disk seam, exactly as internal/modver and internal/outdated keep
// their logic I/O-free.
package update

import (
	"sort"

	"github.com/RomanAgaltsev/keel/internal/lock"
)

// Class is how one file should be treated on update.
type Class int

const (
	// Clean: the user never edited the file (on-disk == recorded original), so the
	// new render is safe to write in place.
	Clean Class = iota
	// Conflict: the user edited the file since scaffolding, so the new render must
	// not overwrite it — the caller writes it beside the original instead.
	Conflict
	// New: a file the updated module renders that does not exist on disk yet.
	New
	// Removed: a file recorded in the lock that the updated module no longer
	// renders. Reported only; never deleted.
	Removed
)

// FileChange is one classified file. Content is the new render (empty for Removed).
type FileChange struct {
	Path    string
	Class   Class
	Content string
}

// Plan is the classified set of changes, sorted by Path for deterministic output.
type Plan struct {
	Changes []FileChange
}

// Input carries everything Classify needs, all supplied by the (impure) caller.
type Input struct {
	// Candidates are the module names to update (version-bumped, or all under
	// --reconfigure), already filtered by any --modules selection.
	Candidates map[string]bool
	// VersionChanged[module] reports whether the module's version actually changed.
	// Used only to decide whether a v1 (hash-less) lock can reconstruct a baseline.
	VersionChanged map[string]bool
	// Render is the freshly rendered dest → content for the full recipe.
	Render map[string]string
	// Owner maps dest → the module that rendered it (render.Plan.Owner()).
	Owner map[string]string
	// Original[module][path] is the sha256 recorded in the lock (empty for v1).
	Original map[string]map[string]string
	// HashOf returns the sha256 of the on-disk file, whether it exists, or an error.
	HashOf func(path string) (sha string, exists bool, err error)
}

// Classify produces the update Plan. See the Class docs for each rule.
func Classify(in Input) (Plan, error) {
	var changes []FileChange

	for dest, content := range in.Render {
		mod := in.Owner[dest]
		if !in.Candidates[mod] {
			continue // only touch files owned by a candidate module
		}
		newHash := hash(content)

		onHash, exists, err := in.HashOf(dest)
		if err != nil {
			return Plan{}, err
		}
		if !exists {
			changes = append(changes, FileChange{Path: dest, Class: New, Content: content})
			continue
		}
		if onHash == newHash {
			continue // file already equals the new render: nothing to do
		}

		origHash, hasOrig := lookupOriginal(in.Original, mod, dest)
		if !hasOrig && !in.VersionChanged[mod] {
			// v1 lock but this module's version didn't change, so the current render
			// reconstructs the original baseline.
			origHash, hasOrig = newHash, true
		}
		switch {
		case hasOrig && onHash == origHash:
			changes = append(changes, FileChange{Path: dest, Class: Clean, Content: content})
		default:
			// Edited by the user, or no baseline to prove otherwise ⇒ conservative.
			changes = append(changes, FileChange{Path: dest, Class: Conflict, Content: content})
		}
	}

	// Removed: recorded for a candidate module but absent from the new render.
	for mod := range in.Candidates {
		for path := range in.Original[mod] {
			if _, rendered := in.Render[path]; !rendered {
				changes = append(changes, FileChange{Path: path, Class: Removed})
			}
		}
	}

	sort.Slice(changes, func(i, j int) bool { return changes[i].Path < changes[j].Path })
	return Plan{Changes: changes}, nil
}

func lookupOriginal(orig map[string]map[string]string, mod, path string) (string, bool) {
	m, ok := orig[mod]
	if !ok {
		return "", false
	}
	h, ok := m[path]
	return h, ok
}

func hash(content string) string { return lock.HashBytes([]byte(content)) }
