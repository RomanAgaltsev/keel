package main

import (
	"fmt"
	"io/fs"
	"path"
	"sort"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert/yaml"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/recipe"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available recipe and modules",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()

			recipes, err := listRecipes()
			if err != nil {
				return err
			}
			fmt.Fprintln(out, "Recipes:")
			for _, r := range recipes {
				fmt.Fprintf(out, "  %-14s %s\n", r.Name, r.Language)
			}

			l := module.NewFSLoader(keel.BuiltinFS)
			names, err := l.ModuleNames()
			if err != nil {
				return err
			}
			fmt.Fprintln(out, "Modules:")
			for _, n := range names {
				m, err := l.Load(n)
				if err != nil {
					return err
				}
				fmt.Fprintf(out, "  %-14s %s\n", m.Name, m.Description)
			}
			return nil
		},
	}
}
func listRecipes() ([]recipe.Recipe, error) {
	entries, err := fs.ReadDir(keel.BuiltinFS, "recipes")
	if err != nil {
		return nil, err
	}
	var out []recipe.Recipe
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, err := fs.ReadFile(keel.BuiltinFS, path.Join("recipes", e.Name()))
		if err != nil {
			return nil, err
		}
		var r recipe.Recipe
		if err := yaml.Unmarshal(b, &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
