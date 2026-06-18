// Package scaffold orchestrates the keel new lifecycle.
package scaffold

import (
	"errors"
	"io/fs"
	"os"
)

// State is the repo-state classification.
type State struct {
	LocalPresent  bool
	RemotePresent bool
}

// localPresent reports whether target exists and is non-empty.
func localPresent(target string) (bool, error) {
	entries, err := os.ReadDir(target)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return len(entries) > 0, nil
}

// pathExists reports whether target exists at all (empty or not). This is
// distinct from localPresent: an existing *empty* dir is not "local present"
// (nothing to adopt) but it does exist, so it must take the overlay write path
// rather than the fresh atomic rename (which refuses any existing target).
func pathExists(target string) (bool, error) {
	_, err := os.Stat(target)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
