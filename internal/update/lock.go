package update

import (
	"sort"

	"github.com/RomanAgaltsev/keel/internal/lock"
)

// NewLock rebuilds the post-update lock. Modules named in refreshed get their new
// version and freshly hashed files (from renderContent, grouped by owner); all
// other modules keep their previous entry. keelVersion stamps the running binary.
func NewLock(old lock.Lock, renderContent, owner map[string]string, refreshed map[string]string, keelVersion string) lock.Lock {
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
	mods := make([]lock.Module, len(old.Modules))
	for i, m := range old.Modules {
		if v, ok := refreshed[m.Name]; ok {
			m.Version = v
			m.Files = filesByModule[m.Name]
		}
		mods[i] = m
	}
	out.Modules = mods
	return out
}
