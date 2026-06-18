package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/prompt"
	"github.com/RomanAgaltsev/keel/internal/provider"
	"github.com/RomanAgaltsev/keel/internal/recipe"
	"github.com/RomanAgaltsev/keel/internal/scaffold"
)

func newNewCmd() *cobra.Command {
	var (
		answersPath string
		noInput     bool
		target      string
		recipeName  string
		remoteURL   string
		overwrite   bool
		dryRun      bool
	)
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Scaffold a new repository from a recipe",
		RunE: func(cmd *cobra.Command, _ []string) error {
			l := module.NewFSLoader(keel.BuiltinFS)

			// Load preset answers (if any).
			preset := answers.Answers{}
			if answersPath != "" {
				p, err := prompt.LoadAnswersFile(answersPath)
				if err != nil {
					return err
				}
				preset = p
			}

			// Resolve the recipe's module list.
			rec, err := recipe.Load(keel.BuiltinFS, recipeName)
			if err != nil {
				return err
			}

			// Merge core + module questions, then collect.
			moduleQs, err := module.RecipeQuestions(l, rec.Modules)
			if err != nil {
				return err
			}
			merged, err := prompt.MergeQuestions(prompt.CoreQuestions(), moduleQs)
			if err != nil {
				return err
			}
			var asker prompt.Asker
			if !noInput {
				asker = prompt.Wizard{}
			}
			ans, err := prompt.Collect(merged, preset, asker)
			if err != nil {
				return err
			}

			if target == "" {
				target = str(ans, "repo_name")
			}

			// Provider selection (Phase 2: only the "none" path; Phase 3 adds github).
			var p provider.Provider
			createRemote, _ := ans["create_remote"].(bool)
			// remoteURL alone is enough to drive clone-then-overlay without a provider.

			res, err := scaffold.Run(cmd.Context(), scaffold.Options{
				Target: target, Recipe: rec.Name, ModuleNames: rec.Modules, Loader: l,
				Provider: p, Answers: ans, CreateRemote: createRemote, RemoteURL: remoteURL,
				Overwrite: overwrite, DryRun: dryRun, KeelVersion: version,
			})
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			verb := "Scaffolded"
			if res.DryRun {
				verb = "[dry-run] would scaffold"
			}
			fmt.Fprintf(out, "%s %q (local=%v, remote=%v)\n", verb, target, res.State.LocalPresent, res.State.RemotePresent)
			fmt.Fprintf(out, "  written: %d, skipped: %d\n", len(res.Written), len(res.Skipped))
			for _, s := range res.NextSteps {
				fmt.Fprintf(out, "  next: %s\n", s)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&answersPath, "answers", "", "answers YAML file")
	cmd.Flags().BoolVar(&noInput, "no-input", false, "never prompt (CI mode)")
	cmd.Flags().StringVar(&target, "target", "", "target directory (default: repo name)")
	cmd.Flags().StringVar(&recipeName, "recipe", "go-service", "recipe to use")
	cmd.Flags().StringVar(&remoteURL, "remote-url", "", "existing remote to clone/wire instead of creating")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "overwrite existing files on overlay")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the plan; touch neither disk nor network")
	return cmd
}

func str(a answers.Answers, k string) string { s, _ := a[k].(string); return s }
