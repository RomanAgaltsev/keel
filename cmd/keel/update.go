package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/lock"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/modver"
	"github.com/RomanAgaltsev/keel/internal/render"
	"github.com/RomanAgaltsev/keel/internal/update"
)

type updateFlags struct {
	path   string
	dryRun bool
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
	return cmd
}

func runUpdate(cmd *cobra.Command, f *updateFlags) error {
	lk, err := lock.Read(filepath.Join(f.path, ".scaffold.lock"))
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

	// Candidate detection: a locked module is a candidate iff its current resolved
	// version is newer than the locked one.
	candidates, versionChanged, err := updateCandidates(lk, comp)
	if err != nil {
		return err
	}

	plan, err := render.BuildRecipe(comp, rec.ModuleNames(), lk.Answers)
	if err != nil {
		return err
	}

	in := update.Input{
		Candidates:     candidates,
		VersionChanged: versionChanged,
		Render:         plan.Files,
		Owner:          plan.Owner(),
		Original:       lockOriginals(lk),
		HashOf:         diskHasher(f.path),
	}
	up, err := update.Classify(in)
	if err != nil {
		return err
	}

	printUpdatePlan(cmd.OutOrStdout(), up, f.dryRun)
	return nil // dry-run only in this slice; apply lands in Plan 2
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
		_, curVer := provenance(comp, m.Name, cur.Version)
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

// provenance returns the source/version the loader reports for a module, falling
// back to the manifest version for a plain builtin loader.
func provenance(comp *module.Composite, name, manifestVer string) (string, string) {
	if p, ok := any(comp).(module.Provenancer); ok {
		return p.Provenance(name)
	}
	return "builtin", manifestVer
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
