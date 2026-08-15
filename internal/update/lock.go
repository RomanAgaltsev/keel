package update

import (
	"sort"

	"github.com/RomanAgaltsev/keel/v2/internal/lock"
)

// NewLock returns the lock to write after an update: refreshed versions and file
// hashes for every module that was rewritten, appended entries for modules the
// recipe gained, and no entry at all for modules it dropped. keelVersion stamps
// the running binary.
//
// The recorded hash is over the bytes keel *rendered* for the new version, not the
// bytes on disk: for a Conflict the user's file is preserved (the new render lands
// in <path>.keel-new), so the lock's baseline tracks "what this version renders" —
// which is what makes a later "accepted the .keel-new" resolve classify as Clean.
func NewLock(old lock.Lock, ms ModuleSet, renderContent, owner map[string]string, keelVersion string) lock.Lock {
	// Pre-group the new renders by module for the refreshed entries.
	filesByModule := map[string][]lock.File{}
	dests := make([]string, 0, len(renderContent))
	for dest := range renderContent {
		dests = append(dests, dest)
	}
	sort.Strings(dests)
	for _, dest := range dests {
		mod := owner[dest]
		filesByModule[mod] = append(filesByModule[mod], lock.File{
			Path:   dest,
			SHA256: lock.HashBytes([]byte(renderContent[dest])),
		})
	}

	out := old
	out.KeelVersion = keelVersion
	added := ms.AddedModules()
	mods := make([]lock.Module, 0, len(old.Modules)+len(added))
	for _, m := range old.Modules {
		if ms.State[m.Name] == Orphaned {
			continue // the recipe no longer names it
		}
		if v, ok := ms.Refreshed[m.Name]; ok {
			m.Version = v
			m.Files = filesByModule[m.Name]
		}
		mods = append(mods, m)
	}
	// Appended in recipe order so the lock stays deterministic across runs.
	for _, name := range added {
		mods = append(mods, lock.Module{
			Name:    name,
			Source:  ms.Source[name],
			Version: ms.Refreshed[name],
			Files:   filesByModule[name],
		})
	}
	out.Modules = mods
	return out
}
