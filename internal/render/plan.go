package render

import (
	"fmt"
	"io/fs"
	"sort"

	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/manifest"
)

// moduleFS pairs a resolved manifest with its template filesystem.
type moduleFS struct {
	Manifest manifest.Manifest
	FS       fs.FS
}

// Plan is the merged set of files to write (dest path -> rendered content).
type Plan struct {
	Files map[string]string
	owner map[string]string // dest -> module name, for collision messages
}

// BuildPlan renders every module in order and merges the results, failing fast
// on any cross-module destination collision.
func BuildPlan(mods []moduleFS, a answers.Answers) (Plan, error) {
	p := Plan{
		Files: map[string]string{},
		owner: map[string]string{},
	}
	for _, mf := range mods {
		files, err := renderModule(mf.Manifest, mf.FS, a)
		if err != nil {
			return Plan{}, err
		}
		dests := make([]string, 0, len(files))
		for d := range files {
			dests = append(dests, d)
		}
		sort.Strings(dests) // deterministic order
		for _, dest := range dests {
			if prev, ok := p.owner[dest]; ok {
				return Plan{}, fmt.Errorf("file collision at %q: modules %q and %q both write it", dest, prev, mf.Manifest.Name)
			}
			p.owner[dest] = mf.Manifest.Name
			p.Files[dest] = files[dest]
		}
	}
	return p, nil
}
