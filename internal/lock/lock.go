// Package lock reads and writes .scaffold.lock - the record of what produced a repo.
package lock

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// currentLockVersion is the schema version Write stamps. v2 added per-file hashes.
const currentLockVersion = 2

// File records one rendered dest and a hash of the bytes keel wrote (not the
// on-disk file, which may later diverge as the user edits it).
type File struct {
	Path   string `yaml:"path"`
	SHA256 string `yaml:"sha256"`
}

// Module records one applied module and its provenance.
type Module struct {
	Name    string `yaml:"name"`
	Source  string `yaml:"source"`
	Version string `yaml:"version"`
	Files   []File `yaml:"files,omitempty"`
}

// Lock is the .scaffold.lock document.
type Lock struct {
	LockVersion int            `yaml:"lock_version"`
	KeelVersion string         `yaml:"keel_version"`
	Recipe      string         `yaml:"recipe"`
	Modules     []Module       `yaml:"modules"`
	Answers     map[string]any `yaml:"answers"`
}

// HashBytes returns the hex sha256 of b, used for the lock's per-file hashes.
func HashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// Write marshals l to path.
func Write(path string, l Lock) error {
	l.LockVersion = currentLockVersion
	b, err := yaml.Marshal(l)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return fmt.Errorf("write lock %q: %w", path, err)
	}
	return nil
}

// Read parses the lock at path.
func Read(path string) (Lock, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Lock{}, fmt.Errorf("read lock %q: %w", path, err)
	}
	var l Lock
	if err := yaml.Unmarshal(b, &l); err != nil {
		return Lock{}, fmt.Errorf("parse lock %q: %w", path, err)
	}
	return l, nil
}
