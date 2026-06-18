// Package lock reads and writes .scaffold.lock - the record of what produced a repo.
package lock

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Module records one applied module and its provenance.
type Module struct {
	Name    string `yaml:"name"`
	Source  string `yaml:"source"`
	Version string `yaml:"version"`
}

// Lock is the .scaffold.lock document.
type Lock struct {
	KeelVersion string         `yaml:"keel_version"`
	Recipe      string         `yaml:"recipe"`
	Modules     []Module       `yaml:"modules"`
	Answers     map[string]any `yaml:"answers"`
}

// Write marshals l to path.
func Write(path string, l Lock) error {
	b, err := yaml.Marshal(l)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
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
