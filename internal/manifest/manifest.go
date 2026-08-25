// Package manifest defines keel's template-module manifest types.
package manifest

import "fmt"

// Manifest describes a single template module (module.yaml).
type Manifest struct {
	Name        string     `yaml:"name"`
	Description string     `yaml:"description"`
	Version     string     `yaml:"version"`
	Language    string     `yaml:"language"` // any | go | rust; enforced against the recipe's language
	Requires    []string   `yaml:"requires"`
	Questions   []Question `yaml:"questions"`
	Files       []FileRule `yaml:"files"`
	Emits       Emits      `yaml:"emits,omitempty"`
	Deps        []Dep      `yaml:"deps,omitempty"`
}

// Question is a single typed prompt contributed by a module.
type Question struct {
	ID       string   `yaml:"id"`
	Prompt   string   `yaml:"prompt"`
	Type     string   `yaml:"type"` // string | bool | select | multiselect | int
	Default  any      `yaml:"default"`
	Options  []string `yaml:"options,omitempty"`
	Required bool     `yaml:"required,omitempty"`
}

// FileRule maps a glob of template files to a destination, optionally gated by When.
type FileRule struct {
	Src  string `yaml:"src"`
	Dest string `yaml:"dest"`
	When string `yaml:"when,omitempty"`
	// UserOwned marks a file whose content becomes the user's once they edit it.
	// go.mod is the motivating case: their first `go get` adds a require line
	// keel did not write.
	//
	// While such a file is untouched it updates normally, so keel keeps
	// actualizing the parts it owns -- `keel update --reconfigure` still
	// rewrites go.mod's module path. Once edited it is skipped silently: no
	// .keel-new sidecar, because a sidecar holding keel's requires *without*
	// the user's would be dangerous to act on. It is never retracted either,
	// and stays recorded in the lock so a later append-merge can reconstruct a
	// baseline.
	UserOwned bool `yaml:"user_owned,omitempty"`
}

// Validate reports whether every contract this manifest declares is well formed.
//
// Called by the loader, so a malformed module.yaml fails when it is read rather
// than rendering something subtly wrong. Emits.Validate existed before this and
// was called by nothing but its own tests -- a guard that cannot fire.
func (m Manifest) Validate() error {
	if err := m.Emits.Validate(); err != nil {
		return fmt.Errorf("module %q: %w", m.Name, err)
	}
	if err := ValidateDeps(m.Deps, m.Language); err != nil {
		return fmt.Errorf("module %q: %w", m.Name, err)
	}
	return nil
}
