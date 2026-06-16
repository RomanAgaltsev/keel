// Package manifest defines keel's template-module and loading.
package manifest

// Manifest describes a single template module (module.yaml).
type Manifest struct {
	Name        string     `yaml:"name"`
	Description string     `yaml:"description"`
	Version     string     `yaml:"version"`
	Language    string     `yaml:"language"` // any | go | rust
	Requires    []string   `yaml:"requires"`
	Questions   []Question `yaml:"questions"`
	Files       []FileRule `yaml:"files"`
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

// FileRule maps a glob of template files to a destionation, optionally gated by When.
type FileRule struct {
	Src  string `yaml:"src"`
	Dest string `yaml:"dest"`
	When string `yaml:"when,omitempty"`
}
