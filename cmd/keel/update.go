package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/git"
	"github.com/RomanAgaltsev/keel/internal/lock"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/render"
	"github.com/RomanAgaltsev/keel/internal/update"
)

type updateFlags struct {
	path        string
	dryRun      bool
	reconfigure bool
	noInput     bool
	commit      bool
	overwrite   bool
	modules     string
	recipe      string
}

func newUpdateCmd() *cobra.Command {
	var f updateFlags
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Re-apply evolved module templates to an existing repo",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUpdate(cmd, &f)
		},
	}
	cmd.Flags().StringVar(&f.path, "path", ".", "repository path to update")
	cmd.Flags().BoolVar(&f.dryRun, "dry-run", false, "print the plan; write nothing (external module sources are still fetched)")
	cmd.Flags().BoolVar(&f.reconfigure, "reconfigure", false, "re-run the wizard and re-render all modules")
	cmd.Flags().BoolVar(&f.noInput, "no-input", false, "never prompt (CI mode); only meaningful with --reconfigure")
	cmd.Flags().BoolVar(&f.commit, "commit", false, "commit the update when there are no conflicts")
	cmd.Flags().BoolVar(&f.overwrite, "overwrite", false, "overwrite user-edited files instead of writing .keel-new")
	cmd.Flags().StringVar(&f.modules, "modules", "", "restrict to a comma-separated subset of modules")
	cmd.Flags().StringVar(&f.recipe, "recipe", "", "recipe to re-apply (overrides the one recorded in .scaffold.lock)")
	return cmd
}

func runUpdate(cmd *cobra.Command, f *updateFlags) error {
	out := cmd.OutOrStdout()
	lockPath := filepath.Join(f.path, ".scaffold.lock")
	lk, err := lock.Read(lockPath)
	if err != nil {
		return err
	}

	rec, recipeDir, _, warning, err := resolveUpdateRecipe(lk, f.path, f.recipe)
	if err != nil {
		return err
	}
	if warning != "" {
		fmt.Fprintln(out, warning)
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

	ans, err := updateAnswers(comp, names, lk, f)
	if err != nil {
		return err
	}

	if f.modules != "" {
		warnUnknownModules(out, lk, splitCSV(f.modules))
	}
	ms, err := update.Resolve(lk, names, compProvenance(comp), update.Options{
		Reconfigure: f.reconfigure,
		Only:        splitCSV(f.modules),
	})
	if err != nil {
		return err
	}

	plan, err := render.BuildRecipe(comp, names, ans)
	if err != nil {
		return err
	}

	up, err := update.Classify(update.Input{
		Candidates:     ms.Candidates(),
		Orphaned:       ms.OrphanedSet(),
		VersionChanged: ms.VersionChanged,
		Render:         plan.Files,
		Owner:          plan.Owner(),
		Original:       lockOriginals(lk),
		HashOf:         diskHasher(f.path),
	})
	if err != nil {
		return err
	}

	if f.dryRun {
		printUpdatePlan(out, up, true, f.overwrite)
		return nil
	}
	// Nothing is behind: leave the lock (and tree) untouched rather than rewriting
	// it just to bump keel_version.
	if len(ms.Candidates()) == 0 && len(ms.OrphanedModules()) == 0 {
		fmt.Fprintln(out, "everything is up to date")
		return nil
	}
	return applyUpdate(cmd, f, lockPath, lk, ans, plan, up, ms)
}

// updateAnswers resolves the answer set used to render. By default the stored
// answers are carried forward, with any questions added since this repo was
// scaffolded filled from their defaults (so an evolved template referencing a new
// key renders instead of failing with a raw missingkey error); a new *required*
// question with no default yields a clear "use --reconfigure" error. Under
// --reconfigure the wizard re-runs, seeded from the lock.
func updateAnswers(comp *module.Composite, names []string, lk lock.Lock, f *updateFlags) (answers.Answers, error) {
	if f.reconfigure {
		return collectAnswers(comp, names, lk.Answers, f.noInput)
	}
	ans, err := collectAnswers(comp, names, lk.Answers, true) // noInput: fill defaults, never prompt
	if err != nil {
		return nil, fmt.Errorf("stored answers are missing a value a newer template needs; re-run with --reconfigure: %w", err)
	}
	return ans, nil
}

// warnUnknownModules notes any --modules name that does not match a module recorded
// in the lock, so a typo isn't silently swallowed by the candidate intersection.
func warnUnknownModules(out io.Writer, lk lock.Lock, requested []string) {
	locked := map[string]bool{}
	for _, m := range lk.Modules {
		locked[m.Name] = true
	}
	for _, name := range requested {
		if !locked[name] {
			fmt.Fprintf(out, "warning: --modules %q is not a module in this repo's lock\n", name)
		}
	}
}

// compProvenance adapts the composite module loader to update.Provenance,
// keeping the resolver free of the loader itself.
func compProvenance(comp *module.Composite) update.Provenance {
	return func(name string) (string, string, error) {
		if _, err := comp.Load(name); err != nil {
			return "", "", err
		}
		source, version := comp.Provenance(name)
		return source, version, nil
	}
}

// applyUpdate writes the classified plan, rewrites the lock to the refreshed
// versions, and commits when --commit is set and there are no conflicts.
func applyUpdate(cmd *cobra.Command, f *updateFlags, lockPath string, lk lock.Lock, ans answers.Answers, plan render.Plan, up update.Plan, ms update.ModuleSet) error {
	out := cmd.OutOrStdout()
	applied, err := update.Apply(up, f.path, f.overwrite)
	if err != nil {
		return err
	}
	newLock := update.NewLock(lk, ms, plan.Files, plan.Owner(), version)
	// Persist the answers actually rendered with — re-collected choices under
	// --reconfigure, or stored answers with newly-added defaults filled — so the
	// lock stays consistent with the hashes just recorded.
	newLock.Answers = ans
	if err := lock.Write(lockPath, newLock); err != nil {
		return err
	}
	printApplied(out, applied)
	reportRemoved(out, applied)

	switch {
	case f.commit && len(applied.Conflicts) == 0:
		// Stage only what keel wrote (plus the lock), never the user's unrelated
		// working-tree changes.
		staged := append([]string{}, applied.Updated...)
		staged = append(staged, applied.New...)
		staged = append(staged, applied.Deleted...)
		staged = append(staged, ".scaffold.lock")
		if err := commitUpdate(cmd.Context(), f.path, lk.Answers, staged); err != nil {
			return err
		}
		fmt.Fprintln(out, "committed: chore: keel update")
	case len(applied.Conflicts) > 0:
		fmt.Fprintln(out, "resolve the .keel-new files, then commit")
	}
	return nil
}

// lockOriginals indexes the lock's recorded hashes as module → path → sha.
func lockOriginals(lk lock.Lock) map[string]map[string]string {
	out := map[string]map[string]string{}
	for _, m := range lk.Modules {
		fm := map[string]string{}
		for _, f := range m.Files {
			fm[f.Path] = f.SHA256
		}
		out[m.Name] = fm
	}
	return out
}

func printUpdatePlan(out io.Writer, up update.Plan, dryRun, overwrite bool) {
	prefix := ""
	if dryRun {
		prefix = "[dry-run] "
	}
	if len(up.Changes) == 0 {
		fmt.Fprintf(out, "%severything is up to date\n", prefix)
		return
	}
	sort.Slice(up.Changes, func(i, j int) bool { return up.Changes[i].Path < up.Changes[j].Path })
	for _, c := range up.Changes {
		// Mirror what Apply would do: with --overwrite an edited file is replaced in
		// place rather than written to a .keel-new sibling.
		label := className(c.Class)
		if overwrite && c.Class == update.Conflict {
			label = "overwrite"
		}
		fmt.Fprintf(out, "%s%-12s %s\n", prefix, label, c.Path)
	}
}

func className(c update.Class) string {
	switch c {
	case update.Clean:
		return "update"
	case update.Conflict:
		return "conflict"
	case update.New:
		return "new"
	case update.Removed:
		return "removed"
	default:
		return "?"
	}
}

// diskHasher returns an update.HashOf seam rooted at repoPath.
func diskHasher(repoPath string) func(string) (string, bool, error) {
	return func(dest string) (string, bool, error) {
		full := filepath.Join(repoPath, filepath.FromSlash(dest))
		b, err := os.ReadFile(full)
		if os.IsNotExist(err) {
			return "", false, nil
		}
		if err != nil {
			return "", false, err
		}
		return lock.HashBytes(b), true, nil
	}
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// commitUpdate stages the given paths and commits. It sets the repo identity from
// the lock's recorded author answers only when both are present, so it never
// clobbers an identity the user already configured.
func commitUpdate(ctx context.Context, path string, answers map[string]any, paths []string) error {
	repo := git.New(path)
	if !repo.IsRepo() {
		return fmt.Errorf("update --commit: %q is not a git repository", path)
	}
	name, _ := answers["author_name"].(string)
	email, _ := answers["author_email"].(string)
	if name != "" && email != "" {
		if err := repo.SetIdentity(ctx, name, email); err != nil {
			return err
		}
	}
	if err := repo.Add(ctx, paths...); err != nil {
		return err
	}
	staged, err := repo.HasStagedChanges(ctx)
	if err != nil {
		return err
	}
	if !staged {
		return nil
	}
	return repo.Commit(ctx, "chore: keel update")
}

// reportRemoved prints the files that keel no longer produces. Apply deliberately
// leaves them on disk - deleting user files on an update is not keel's call - so
// the report has to be actionable, or a consolidated workflow silently runs
// alongside the four files it replaced.
func reportRemoved(w io.Writer, a update.Applied) {
	if len(a.Kept) == 0 {
		return
	}
	fmt.Fprintf(w, "\n%d file(s) are no longer produced by keel and were left in place:\n\n", len(a.Kept))
	// Apply sorts these (apply.go:56); sort a copy anyway so the output is
	// deterministic for any caller, matching printApplied.
	removed := append([]string{}, a.Kept...)
	sort.Strings(removed)
	for _, p := range removed {
		fmt.Fprintf(w, "    rm %s\n", p)
	}
	fmt.Fprint(w, "\nReview them, then delete the ones you no longer want.\n")
}

// printApplied prints the per-class summary, deterministically.
func printApplied(out io.Writer, a update.Applied) {
	line := func(label string, items []string) {
		if len(items) == 0 {
			return
		}
		sort.Strings(items)
		for _, p := range items {
			fmt.Fprintf(out, "%-9s %s\n", label, p)
		}
	}
	line("updated", a.Updated)
	line("new", a.New)
	line("conflict", a.Conflicts)
	line("removed", a.Kept)
	fmt.Fprintf(out, "updated %d, new %d, conflicts %d, removed %d\n",
		len(a.Updated), len(a.New), len(a.Conflicts), len(a.Kept))
}
