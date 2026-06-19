package render

import (
	"os"
	"path/filepath"
	"sort"
)

// WriteResult records what an overlay write did.
type WriteResult struct {
	Written []string // dest paths newly written (sorted)
	Skipped []string // dest paths kept because they already existed (sorted)
}

// OverlayPlan writes the plan's files into an existing directory. By default it
// skips destinations that already exist (keeping the user's file). Overwrite=true
// replaces them. It never deletes files. Parent dirs are created as needed.
func OverlayPlan(p Plan, target string, overwrite bool) (WriteResult, error) {
	var res WriteResult
	dests := make([]string, 0, len(p.Files))
	for d := range p.Files {
		dests = append(dests, d)
	}
	sort.Strings(dests)

	for _, dest := range dests {
		full := filepath.Join(target, filepath.FromSlash(dest))
		if _, err := os.Stat(full); err == nil && !overwrite {
			res.Skipped = append(res.Skipped, dest)
			continue
		} else if err != nil && !os.IsNotExist(err) {
			return WriteResult{}, err
		}
		//nolint:gosec // scaffolded project dirs are intended to be world-readable
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return WriteResult{}, err
		}
		//nolint:gosec // scaffolded project files are intended to be world-readable
		if err := os.WriteFile(full, []byte(p.Files[dest]), 0o644); err != nil {
			return WriteResult{}, err
		}
		res.Written = append(res.Written, dest)
	}
	return res, nil
}
