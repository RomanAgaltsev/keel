package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/prompt"
	"github.com/RomanAgaltsev/keel/internal/provider"
	"github.com/RomanAgaltsev/keel/internal/recipe"
	"github.com/RomanAgaltsev/keel/internal/scaffold"
)

// newFlags holds the parsed flags for the new command.
type newFlags struct {
	answersPath string
	noInput     bool
	target      string
	recipeName  string
	remoteURL   string
	overwrite   bool
	dryRun      bool
}

func newNewCmd() *cobra.Command {
	var f newFlags
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Scaffold a new repository from a recipe",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runNew(cmd, &f)
		},
	}
	cmd.Flags().StringVar(&f.answersPath, "answers", "", "answers YAML file")
	cmd.Flags().BoolVar(&f.noInput, "no-input", false, "never prompt (CI mode)")
	cmd.Flags().StringVar(&f.target, "target", "", "target directory (default: repo name)")
	cmd.Flags().StringVar(&f.recipeName, "recipe", "go-service", "recipe to use")
	cmd.Flags().StringVar(&f.remoteURL, "remote-url", "", "existing remote to clone/wire instead of creating")
	cmd.Flags().BoolVar(&f.overwrite, "overwrite", false, "overwrite existing files on overlay")
	cmd.Flags().BoolVar(&f.dryRun, "dry-run", false, "print the plan; touch neither disk nor network")
	return cmd
}

func runNew(cmd *cobra.Command, f *newFlags) error {
	l := module.NewFSLoader(keel.BuiltinFS)

	// Load preset answers (if any).
	preset := answers.Answers{}
	if f.answersPath != "" {
		p, err := prompt.LoadAnswersFile(f.answersPath)
		if err != nil {
			return err
		}
		preset = p
	}

	// Resolve the recipe's module list.
	rec, err := recipe.Load(keel.BuiltinFS, f.recipeName)
	if err != nil {
		return err
	}

	ans, err := collectAnswers(l, rec, preset, f.noInput)
	if err != nil {
		return err
	}

	target := f.target
	if target == "" {
		target = str(ans, "repo_name")
	}

	// Provider selection (Phase 2: only the "none" path; Phase 3 adds github).
	var p provider.Provider
	createRemote, _ := ans["create_remote"].(bool)
	// remoteURL alone is enough to drive clone-then-overlay without a provider.

	res, err := scaffold.Run(cmd.Context(), scaffold.Options{
		Target: target, Recipe: rec.Name, ModuleNames: rec.Modules, Loader: l,
		Provider: p, Answers: ans, CreateRemote: createRemote, RemoteURL: f.remoteURL,
		Overwrite: f.overwrite, DryRun: f.dryRun, KeelVersion: version,
	})
	if err != nil {
		return err
	}

	printResult(cmd.OutOrStdout(), target, res)
	return nil
}

// collectAnswers merges core + module questions and collects answers.
func collectAnswers(l module.Loader, rec recipe.Recipe, preset answers.Answers, noInput bool) (answers.Answers, error) {
	moduleQs, err := module.RecipeQuestions(l, rec.Modules)
	if err != nil {
		return nil, err
	}
	merged, err := prompt.MergeQuestions(prompt.CoreQuestions(), moduleQs)
	if err != nil {
		return nil, err
	}
	var asker prompt.Asker
	if !noInput {
		asker = prompt.Wizard{}
	}
	return prompt.Collect(merged, preset, asker)
}

func printResult(out io.Writer, target string, res scaffold.Result) {
	verb := "Scaffolded"
	if res.DryRun {
		verb = "[dry-run] would scaffold"
	}
	fmt.Fprintf(out, "%s %q (local=%v, remote=%v)\n", verb, target, res.State.LocalPresent, res.State.RemotePresent)
	fmt.Fprintf(out, "  written: %d, skipped: %d\n", len(res.Written), len(res.Skipped))
	for _, s := range res.NextSteps {
		fmt.Fprintf(out, "  next: %s\n", s)
	}
}

func str(a answers.Answers, k string) string { s, _ := a[k].(string); return s }
