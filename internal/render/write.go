package render

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// WritePlan writes the plan atomically: it renders into a temp dir beside the
// target, then renames it into place. It errors if target already exists.
func WritePlan(p Plan, target string) error {
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("target %q already exists", target)
	} else if !os.IsNotExist(err) {
		return err
	}

	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(parent, ".keel-tmp-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp) // no-op after a successful rename

	dests := make([]string, 0, len(p.Files))
	for d := range p.Files {
		dests = append(dests, d)
	}
	sort.Strings(dests)
	for _, dest := range dests {
		full := filepath.Join(tmp, filepath.FromSlash(dest))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, []byte(p.Files[dest]), 0o644); err != nil {
			return err
		}
	}

	if err := os.Rename(tmp, target); err != nil {
		return fmt.Errorf("finalize %q: %w", target, err)
	}
	return nil
}
