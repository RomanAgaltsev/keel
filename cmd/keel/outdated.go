package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/RomanAgaltsev/keel/v2"
	"github.com/RomanAgaltsev/keel/v2/internal/lock"
	"github.com/RomanAgaltsev/keel/v2/internal/module"
	"github.com/RomanAgaltsev/keel/v2/internal/outdated"
	"github.com/RomanAgaltsev/keel/v2/internal/update"
)

// errUpdatesAvailable signals a non-zero exit without being a real failure.
var errUpdatesAvailable = errors.New("updates available")

func newOutdatedCmd() *cobra.Command {
	var (
		path        string
		toolsOnly   bool
		modulesOnly bool
	)
	cmd := &cobra.Command{
		Use:   "outdated",
		Short: "Report outdated tool/action pins and keel modules (read-only)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runOutdated(cmd, path, toolsOnly, modulesOnly)
		},
	}
	cmd.Flags().StringVar(&path, "path", ".", "repository path to inspect")
	cmd.Flags().BoolVar(&toolsOnly, "tools-only", false, "check only tool/action pins")
	cmd.Flags().BoolVar(&modulesOnly, "modules-only", false, "check only keel module versions")
	return cmd
}

func runOutdated(cmd *cobra.Command, path string, toolsOnly, modulesOnly bool) error {
	out := cmd.OutOrStdout()
	var rep outdated.Report

	if !toolsOnly {
		mods, added, orphaned, skipped, err := moduleUpdates(path)
		if err != nil {
			return err
		}
		rep.Modules = mods
		rep.AddedModules = added
		rep.OrphanedModules = orphaned
		rep.Skipped = append(rep.Skipped, skipped...)
	}
	if !modulesOnly {
		tools, skipped, err := toolUpdates(cmd, path)
		if err != nil {
			return err
		}
		rep.Tools = tools
		if skipped > 0 {
			fmt.Fprintf(out, "(%d pin(s) skipped: no release or rate-limited)\n", skipped)
		}
	}

	printReport(out, rep)

	// Three outcomes, three codes — the same split `keel settings apply --check`
	// uses, and for the same reason: CI must be able to tell "this repo is behind"
	// from "keel could not look". Collapsing them is how `outdated` came to report
	// success for a directory it never found a lock in.
	switch {
	case len(rep.Skipped) > 0:
		return exitCodeError{code: 2, err: errors.New("one or more checks could not run")}
	case !rep.Clean():
		return errUpdatesAvailable
	default:
		return nil
	}
}

// moduleUpdates reports version-behind modules plus the composition drift that
// `keel update` would act on, so the two commands never disagree.
// moduleUpdates reports version-behind modules plus the composition drift that
// `keel update` would act on, so the two commands never disagree.
//
// It degrades rather than failing — a read-only command should say what it can —
// but every degradation is recorded in the report. Returning a clean report for a
// check that never ran is how `outdated` came to answer "Everything is up to
// date" for a directory it had not even found a lock in.
func moduleUpdates(path string) (ups []outdated.ModuleUpdate, added, orphaned []string, skipped []outdated.SkippedCheck, err error) {
	lk, lerr := lock.Read(filepath.Join(path, ".scaffold.lock"))
	if lerr != nil {
		return nil, nil, nil, []outdated.SkippedCheck{{
			What:   "modules",
			Reason: fmt.Sprintf("no .scaffold.lock in %q — not a keel-scaffolded repo, or the wrong --path", path),
		}}, nil
	}
	l := module.NewFSLoader(keel.BuiltinFS)
	ups, err = outdated.ModuleUpdates(l, lk.Modules)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	added, orphaned, skipped = compositionDrift(lk, path)
	return ups, added, orphaned, skipped, nil
}

// compositionDrift reports the modules the recipe has gained or dropped. Any step
// it cannot complete becomes a SkippedCheck rather than a silent empty result.
func compositionDrift(lk lock.Lock, path string) (added, orphaned []string, skipped []outdated.SkippedCheck) {
	skip := func(reason string) ([]string, []string, []outdated.SkippedCheck) {
		return nil, nil, []outdated.SkippedCheck{{What: "composition", Reason: reason}}
	}

	rec, _, _, _, err := resolveUpdateRecipe(lk, path, "")
	if err != nil {
		return skip(fmt.Sprintf("recipe %q could not be resolved: %v", lk.Recipe, err))
	}
	// Builtin-only composite: `outdated` never fetches external sources, so a
	// recipe naming one cannot be resolved here — which is reported, not hidden.
	comp, err := module.NewComposite(keel.BuiltinFS, nil)
	if err != nil {
		return skip(fmt.Sprintf("module loader: %v", err))
	}
	ms, err := update.Resolve(lk, rec.ModuleNames(), compProvenance(comp), update.Options{})
	if err != nil {
		return skip(fmt.Sprintf("%v — run `keel update --dry-run` for the full picture", err))
	}
	return ms.AddedModules(), ms.OrphanedModules(), nil
}

func toolUpdates(cmd *cobra.Command, path string) ([]outdated.ToolUpdate, int, error) {
	files, err := readPinFiles(path)
	if err != nil {
		return nil, 0, err
	}
	pins := outdated.ParsePins(files)
	rc := outdated.NewGitHubReleases(firstEnv("KEEL_GITHUB_TOKEN", "GITHUB_TOKEN"))
	ups, skipped := outdated.ToolUpdates(cmd.Context(), pins, rc)
	return ups, skipped, nil
}

// readPinFiles loads Taskfile.yml and every .github/workflows/*.yml under path.
func readPinFiles(path string) (map[string][]byte, error) {
	files := map[string][]byte{}
	if b, err := os.ReadFile(filepath.Join(path, "Taskfile.yml")); err == nil {
		files["Taskfile.yml"] = b
	}
	wfDir := filepath.Join(path, ".github", "workflows")
	entries, err := os.ReadDir(wfDir)
	if errors.Is(err, fs.ErrNotExist) {
		return files, nil
	}
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || (!strings.HasSuffix(e.Name(), ".yml") && !strings.HasSuffix(e.Name(), ".yaml")) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(wfDir, e.Name()))
		if err != nil {
			return nil, err
		}
		files[filepath.ToSlash(filepath.Join(".github/workflows", e.Name()))] = b
	}
	return files, nil
}

func printReport(out io.Writer, rep outdated.Report) {
	// Only a report where every check actually ran may claim the repo is current.
	if rep.Clean() && len(rep.Skipped) == 0 {
		fmt.Fprintln(out, "Everything is up to date.")
		return
	}
	for _, s := range rep.Skipped {
		fmt.Fprintf(out, "not checked (%s): %s\n", s.What, s.Reason)
	}
	if rep.Clean() {
		fmt.Fprintln(out, "Everything that was checked is up to date.")
	}
	if len(rep.Tools) > 0 {
		fmt.Fprintln(out, "Outdated tools/actions:")
		for _, u := range rep.Tools {
			fmt.Fprintf(out, "  %-32s %s -> %s\n", u.Repo, u.Current, u.Latest)
		}
	}
	if len(rep.Modules) > 0 {
		fmt.Fprintln(out, "Outdated keel modules (re-scaffold or bump the pins to update):")
		for _, u := range rep.Modules {
			fmt.Fprintf(out, "  %-32s %s -> %s\n", u.Name, u.Current, u.Latest)
		}
	}
	for _, name := range rep.AddedModules {
		fmt.Fprintf(out, "module   %-24s not in this repo (added to the recipe)\n", name)
	}
	for _, name := range rep.OrphanedModules {
		fmt.Fprintf(out, "module   %-24s no longer in the recipe\n", name)
	}
}
