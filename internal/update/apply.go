package update

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/RomanAgaltsev/keel/internal/render"
)

// Applied reports what Apply wrote, each list sorted by path.
type Applied struct {
	Updated   []string // Clean (or overwritten Conflict) written in place
	New       []string // newly created files
	Conflicts []string // user-edited files preserved; <path>.keel-new written
	Removed   []string // recorded in the lock, no longer rendered (left on disk)
}

// Apply writes the classified plan under target. Clean/New are written in place;
// a Conflict preserves the user's file and writes the new render to
// <path>.keel-new unless overwrite is set; Removed files are left untouched.
func Apply(p Plan, target string, overwrite bool) (Applied, error) {
	var a Applied
	for _, c := range p.Changes {
		switch c.Class {
		case New:
			if err := writeFile(target, c.Path, c.Content); err != nil {
				return Applied{}, err
			}
			a.New = append(a.New, c.Path)
		case Clean:
			if err := writeFile(target, c.Path, c.Content); err != nil {
				return Applied{}, err
			}
			a.Updated = append(a.Updated, c.Path)
		case Conflict:
			if overwrite {
				if err := writeFile(target, c.Path, c.Content); err != nil {
					return Applied{}, err
				}
				a.Updated = append(a.Updated, c.Path)
				continue
			}
			if err := writeFile(target, c.Path+".keel-new", c.Content); err != nil {
				return Applied{}, err
			}
			a.Conflicts = append(a.Conflicts, c.Path)
		case Removed:
			a.Removed = append(a.Removed, c.Path)
		}
	}
	sort.Strings(a.Updated)
	sort.Strings(a.New)
	sort.Strings(a.Conflicts)
	sort.Strings(a.Removed)
	return a, nil
}

// writeFile writes content to target/dest, creating parent dirs, after guarding
// that dest does not escape the target tree.
func writeFile(target, dest, content string) error {
	if err := render.SafeDest(dest); err != nil {
		return err
	}
	full := filepath.Join(target, filepath.FromSlash(dest))
	//nolint:gosec // scaffolded project dirs are intended to be world-readable
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	//nolint:gosec // scaffolded project files are intended to be world-readable
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %q: %w", dest, err)
	}
	return nil
}
