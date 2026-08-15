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
	Deleted   []string // no longer produced by the recipe, and untouched — removed
	Kept      []string // no longer produced, but user-edited — left on disk
}

// Apply writes the classified plan under target. Clean/New are written in place;
// a Conflict preserves the user's file and writes the new render to
// <path>.keel-new unless overwrite is set; Removed files are deleted;
// RemovedEdited files are left untouched and reported.
func Apply(p Plan, target string, overwrite bool) (Applied, error) {
	var a Applied
	for _, c := range p.Changes {
		if err := applyChange(&a, c, target, overwrite); err != nil {
			return Applied{}, err
		}
	}
	sort.Strings(a.Updated)
	sort.Strings(a.New)
	sort.Strings(a.Conflicts)
	sort.Strings(a.Deleted)
	sort.Strings(a.Kept)
	return a, nil
}

// applyChange performs one classified change and records it in a.
func applyChange(a *Applied, c FileChange, target string, overwrite bool) error {
	switch c.Class {
	case New:
		if err := writeFile(target, c.Path, c.Content); err != nil {
			return err
		}
		a.New = append(a.New, c.Path)
	case Clean:
		if err := writeFile(target, c.Path, c.Content); err != nil {
			return err
		}
		a.Updated = append(a.Updated, c.Path)
	case Conflict:
		return applyConflict(a, c, target, overwrite)
	case Removed:
		if err := removeFile(target, c.Path); err != nil {
			return err
		}
		a.Deleted = append(a.Deleted, c.Path)
	case RemovedEdited:
		a.Kept = append(a.Kept, c.Path)
	}
	return nil
}

// applyConflict preserves the user's edited file and writes the new render to
// <path>.keel-new, unless --overwrite says to replace it in place.
func applyConflict(a *Applied, c FileChange, target string, overwrite bool) error {
	if overwrite {
		if err := writeFile(target, c.Path, c.Content); err != nil {
			return err
		}
		a.Updated = append(a.Updated, c.Path)
		return nil
	}
	if err := writeFile(target, c.Path+".keel-new", c.Content); err != nil {
		return err
	}
	a.Conflicts = append(a.Conflicts, c.Path)
	return nil
}

// removeFile deletes target/dest after the same escape guard writeFile uses. A
// file that is already gone is success: the caller's intent was its absence.
func removeFile(target, dest string) error {
	if err := render.SafeDest(dest); err != nil {
		return err
	}
	full := filepath.Join(target, filepath.FromSlash(dest))
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove %q: %w", dest, err)
	}
	return nil
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
