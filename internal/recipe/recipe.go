// Package recipe defines a recipe: a named composition of modules.
package recipe

import (
	"fmt"
	"io/fs"
	"os"
	"path"

	"gopkg.in/yaml.v3"
)

// Recipe composes modules into a scaffoldable project type.
type Recipe struct {
	Name     string      `yaml:"name"`
	Language string      `yaml:"language"`
	Modules  []ModuleRef `yaml:"modules"`
}

// ModuleNames returns the module names in recipe order.
func (r Recipe) ModuleNames() []string {
	out := make([]string, len(r.Modules))
	for i, m := range r.Modules {
		out[i] = m.Name
	}
	return out
}

// ModuleRef is a recipe entry: a builtin module (Source nil) or an external one.
type ModuleRef struct {
	Name   string
	Source *Source
}

// UnmarshalYAML accepts either a scalar (builtin module name) or a mapping
// {name, source}.
func (r *ModuleRef) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		r.Name = n.Value
		return nil
	}
	var m struct {
		Name   string  `yaml:"name"`
		Source *Source `yaml:"source"`
	}
	if err := n.Decode(&m); err != nil {
		return err
	}
	if m.Name == "" {
		return fmt.Errorf("recipe module entry is missing a name")
	}
	if m.Source != nil {
		if err := m.Source.validate(); err != nil {
			return fmt.Errorf("module %q: %w", m.Name, err)
		}
	}
	r.Name, r.Source = m.Name, m.Source
	return nil
}

// Source locates an external module. Exactly one of Dir or Git must be set.
type Source struct {
	Dir    string `yaml:"dir,omitempty"`
	Git    string `yaml:"git,omitempty"` // repository URL
	Subdir string `yaml:"subdir,omitempty"`
	Ref    string `yaml:"ref,omitempty"` // tag / branch / commit
}

func (s Source) validate() error {
	switch {
	case s.Dir != "" && s.Git != "":
		return fmt.Errorf("source has both dir and git")
	case s.Dir == "" && s.Git == "":
		return fmt.Errorf("source has neither dir nor git")
	}
	return nil
}

// Load reads recipes/<name>.yaml from fsys.
func Load(fsys fs.FS, name string) (Recipe, error) {
	b, err := fs.ReadFile(fsys, path.Join("recipes", name+".yaml"))
	if err != nil {
		return Recipe{}, fmt.Errorf("load recipe %q: %w", name, err)
	}
	var r Recipe
	if err := yaml.Unmarshal(b, &r); err != nil {
		return Recipe{}, fmt.Errorf("parse recipe %q: %w", name, err)
	}
	return r, nil
}

// LoadFile reads a recipe from a YAML file on disk (user-supplied recipes).
func LoadFile(path string) (Recipe, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Recipe{}, fmt.Errorf("read recipe %q: %w", path, err)
	}
	var r Recipe
	if err := yaml.Unmarshal(b, &r); err != nil {
		return Recipe{}, fmt.Errorf("parse recipe %q: %w", path, err)
	}
	return r, nil
}
