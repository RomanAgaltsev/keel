package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/git"
	"github.com/RomanAgaltsev/keel/internal/lock"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/modver"
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
	cmd.Flags().BoolVar(&f.dryRun, "dry-run", false, "print the plan; write nothing")
	cmd.Flags().BoolVar(&f.reconfigure, "reconfigure", false, "re-run the wizard and re-render all modules")
	cmd.Flags().BoolVar(&f.noInput, "no-input", false, "never prompt (CI mode); only meaningful with --reconfigure")
	cmd.Flags().BoolVar(&f.commit, "commit", false, "commit the update when there are no conflicts")
	cmd.Flags().BoolVar(&f.overwrite, "overwrite", false, "overwrite user-edited files instead of writing .keel-new")
	cmd.Flags().StringVar(&f.modules, "modules", "", "restrict to a comma-separated subset of modules")
	return cmd
}

func runUpdate(cmd *cobra.Command, f *updateFlags) error {
	lockPath := filepath.Join(f.path, ".scaffold.lock")
	lk, err := lock.Read(lockPath)
	if err != nil {
		return err
	}

	rec, recipeDir, err := loadRecipe(lk.Recipe)
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

	// Answers: stored verbatim by default; re-collected under --reconfigure.
	ans := lk.Answers
	if f.reconfigure {
		ans, err = collectAnswers(comp, names, lk.Answers, f.noInput)
		if err != nil {
			return err
		}
	}

	candidates, versionChanged, refreshed, err := selectCandidates(f, lk, comp)
	if err != nil {
		return err
	}

	plan, err := render.BuildRecipe(comp, names, ans)
	if err != nil {
		return err
	}

	up, err := update.Classify(update.Input{
		Candidates:     candidates,
		VersionChanged: versionChanged,
		Render:         plan.Files,
		Owner:          plan.Owner(),
		Original:       lockOriginals(lk),
		HashOf:         diskHasher(f.path),
	})
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if f.dryRun {
		printUpdatePlan(out, up, true)
		return nil
	}
	return applyUpdate(cmd, f, lockPath, lk, plan, up, refreshed)
}

// selectCandidates determines which modules to update (version-bumped only, or
// all under --reconfigure, intersected with --modules) and the version each
// candidate should be refreshed to.
func selectCandidates(f *updateFlags, lk lock.Lock, comp *module.Composite) (candidates, versionChanged map[string]bool, refreshed map[string]string, err error) {
	candidates, versionChanged, err = updateCandidates(lk, comp)
	if err != nil {
		return nil, nil, nil, err
	}
	if f.reconfigure {
		candidates = allModules(lk) // reconfigure touches every recorded module
	}
	if f.modules != "" {
		candidates = intersect(candidates, splitCSV(f.modules))
	}
	refreshed, err = refreshedVersions(candidates, comp)
	if err != nil {
		return nil, nil, nil, err
	}
	return candidates, versionChanged, refreshed, nil
}

// applyUpdate writes the classified plan, rewrites the lock to the refreshed
// versions, and commits when --commit is set and there are no conflicts.
func applyUpdate(cmd *cobra.Command, f *updateFlags, lockPath string, lk lock.Lock, plan render.Plan, up update.Plan, refreshed map[string]string) error {
	out := cmd.OutOrStdout()
	applied, err := update.Apply(up, f.path, f.overwrite)
	if err != nil {
		return err
	}
	newLock := update.NewLock(lk, plan.Files, plan.Owner(), refreshed, version)
	if err := lock.Write(lockPath, newLock); err != nil {
		return err
	}
	printApplied(out, applied)

	switch {
	case f.commit && len(applied.Conflicts) == 0:
		if err := commitUpdate(cmd.Context(), f.path, lk.Answers); err != nil {
			return err
		}
		fmt.Fprintln(out, "committed: chore: keel update")
	case len(applied.Conflicts) > 0:
		fmt.Fprintln(out, "resolve the .keel-new files, then commit")
	}
	return nil
}

// updateCandidates returns the set of modules whose current version is ahead of
// the locked version, plus a version-changed flag per locked module.
func updateCandidates(lk lock.Lock, comp *module.Composite) (cand, changed map[string]bool, err error) {
	cand, changed = map[string]bool{}, map[string]bool{}
	for _, m := range lk.Modules {
		cur, lerr := comp.Load(m.Name)
		if lerr != nil {
			return nil, nil, fmt.Errorf("module %q: %w", m.Name, lerr)
		}
		curVer := resolvedVersion(comp, m.Name, cur.Version)
		cmp, cerr := modver.Compare(m.Version, curVer)
		if cerr != nil {
			// Unparseable versions ⇒ treat as unchanged (don't guess a bump).
			continue
		}
		if cmp < 0 {
			cand[m.Name] = true
		}
		changed[m.Name] = cmp != 0
	}
	return cand, changed, nil
}

// resolvedVersion returns the version the loader reports for a module, falling
// back to the manifest version for a plain builtin loader.
func resolvedVersion(comp *module.Composite, name, manifestVer string) string {
	if p, ok := any(comp).(module.Provenancer); ok {
		_, ver := p.Provenance(name)
		return ver
	}
	return manifestVer
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

func printUpdatePlan(out interface{ Write([]byte) (int, error) }, up update.Plan, dryRun bool) {
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
		fmt.Fprintf(out, "%s%-12s %s\n", prefix, className(c.Class), c.Path)
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

// allModules returns every recorded module as a candidate set.
func allModules(lk lock.Lock) map[string]bool {
	out := map[string]bool{}
	for _, m := range lk.Modules {
		out[m.Name] = true
	}
	return out
}

// refreshedVersions maps each candidate module to its current resolved version.
func refreshedVersions(candidates map[string]bool, comp *module.Composite) (map[string]string, error) {
	out := map[string]string{}
	for name := range candidates {
		m, err := comp.Load(name)
		if err != nil {
			return nil, fmt.Errorf("module %q: %w", name, err)
		}
		out[name] = resolvedVersion(comp, name, m.Version)
	}
	return out, nil
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

func intersect(set map[string]bool, keep []string) map[string]bool {
	want := map[string]bool{}
	for _, k := range keep {
		want[k] = true
	}
	out := map[string]bool{}
	for k := range set {
		if want[k] {
			out[k] = true
		}
	}
	return out
}

// commitUpdate stages everything and commits, setting identity from the lock's
// recorded author answers (the repo may not have a local identity configured).
func commitUpdate(ctx context.Context, path string, answers map[string]any) error {
	repo := git.New(path)
	if !repo.IsRepo() {
		return fmt.Errorf("update --commit: %q is not a git repository", path)
	}
	name, _ := answers["author_name"].(string)
	email, _ := answers["author_email"].(string)
	if err := repo.SetIdentity(ctx, name, email); err != nil {
		return err
	}
	if err := repo.AddAll(ctx); err != nil {
		return err
	}
	changed, err := repo.HasChanges(ctx)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	return repo.Commit(ctx, "chore: keel update")
}

// printApplied prints the per-class summary, deterministically.
func printApplied(out interface{ Write([]byte) (int, error) }, a update.Applied) {
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
	line("removed", a.Removed)
	fmt.Fprintf(out, "updated %d, new %d, conflicts %d, removed %d\n",
		len(a.Updated), len(a.New), len(a.Conflicts), len(a.Removed))
}
