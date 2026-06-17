// Package recipe defines a recipe: a named composition of modules.
package recipe

// Recipe composes modules into a scaffoldable project type.
type Recipe struct {
	Name     string   `yaml:"name"`
	Language string   `yaml:"language"`
	Modules  []string `yaml:"modules"`
}
