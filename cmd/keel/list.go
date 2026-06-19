package main

import (
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/recipe"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available recipes and modules",
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
	out := make([]recipe.Recipe, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".yaml")
		r, err := recipe.Load(keel.BuiltinFS, name)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
