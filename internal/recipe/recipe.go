// Package recipe defines a recipe: a named composition of modules.
package recipe

import (
	"fmt"
	"io/fs"
	"path"

	"gopkg.in/yaml.v3"
)

// Recipe composes modules into a scaffoldable project type.
type Recipe struct {
	Name     string   `yaml:"name"`
	Language string   `yaml:"language"`
	Modules  []string `yaml:"modules"`
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
