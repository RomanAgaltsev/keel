package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/prompt"
	"github.com/RomanAgaltsev/keel/internal/provider"
	"github.com/RomanAgaltsev/keel/internal/recipe"
	"github.com/RomanAgaltsev/keel/internal/scaffold"
	"github.com/RomanAgaltsev/keel/internal/source"
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
	// Load preset answers (if any).
	preset := answers.Answers{}
	if f.answersPath != "" {
		p, err := prompt.LoadAnswersFile(f.answersPath)
		if err != nil {
			return err
		}
		preset = p
	}

	// Resolve the recipe (builtin name or a file path) and any external sources.
	rec, recipeDir, err := loadRecipe(f.recipeName)
	if err != nil {
		return err
	}
	externals, err := resolveExternals(cmd.Context(), rec, recipeDir)
	if err != nil {
		return err
	}
	comp, err := module.NewComposite(keel.BuiltinFS, externals)
	if err != nil {
		return err
	}
	names := rec.ModuleNames()

	ans, err := collectAnswers(comp, names, preset, f.noInput)
	if err != nil {
		return err
	}

	target := f.target
	if target == "" {
		target = ans.String("repo_name")
	}

	// Provider selection.
	createRemote := ans.Bool("create_remote")
	providerName := ans.String("provider")
	var p provider.Provider
	if createRemote && f.remoteURL == "" {
		var err error
		p, err = provider.For(providerName, provider.Env{
			Token: firstEnv("KEEL_GITHUB_TOKEN", "GITHUB_TOKEN"),
			Owner: ownerOrEnv(ans.String("module_path")),
		})
		if err != nil {
			return err
		}
	}

	res, err := scaffold.Run(cmd.Context(), scaffold.Options{
		Target: target, Recipe: rec.Name, ModuleNames: names, Loader: comp,
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
func collectAnswers(l module.Loader, names []string, preset answers.Answers, noInput bool) (answers.Answers, error) {
	moduleQs, err := module.RecipeQuestions(l, names)
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

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

// ownerFromModulePath extracts the owner from a module path like
// "github.com/Owner/repo" → "Owner". Empty if it can't be determined.
func ownerFromModulePath(mp string) string {
	parts := strings.Split(mp, "/")
	if len(parts) >= 3 {
		return parts[1]
	}
	return ""
}

// ownerOrEnv prefers an explicit $KEEL_GITHUB_OWNER, else derives the owner
// from the module path.
func ownerOrEnv(modulePath string) string {
	if v := firstEnv("KEEL_GITHUB_OWNER"); v != "" {
		return v
	}
	return ownerFromModulePath(modulePath)
}

// loadRecipe loads a recipe by builtin name or from a file path. The returned
// recipeDir is the base for resolving relative dir sources (empty for builtin).
func loadRecipe(nameOrPath string) (recipe.Recipe, string, error) {
	if isRecipeFile(nameOrPath) {
		rec, err := recipe.LoadFile(nameOrPath)
		return rec, filepath.Dir(nameOrPath), err
	}
	rec, err := recipe.Load(keel.BuiltinFS, nameOrPath)
	return rec, "", err
}

func isRecipeFile(s string) bool {
	if strings.HasSuffix(s, ".yaml") || strings.HasSuffix(s, ".yml") {
		return true
	}
	info, err := os.Stat(s)
	return err == nil && !info.IsDir()
}

// resolveExternals fetches/locates every source-bearing module ref.
func resolveExternals(ctx context.Context, rec recipe.Recipe, recipeDir string) ([]module.External, error) {
	var ext []module.External
	for _, m := range rec.Modules {
		if m.Source == nil {
			continue
		}
		res, err := source.Resolve(ctx, *m.Source, recipeDir)
		if err != nil {
			return nil, fmt.Errorf("module %q: %w", m.Name, err)
		}
		ext = append(ext, module.External{Name: m.Name, FS: res.FS, Source: res.Source, Version: res.Version})
	}
	return ext, nil
}
