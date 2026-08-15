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

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/lock"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/outdated"
	"github.com/RomanAgaltsev/keel/internal/update"
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
		mods, added, orphaned, err := moduleUpdates(path)
		if err != nil {
			return err
		}
		rep.Modules = mods
		rep.AddedModules = added
		rep.OrphanedModules = orphaned
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
	if !rep.Empty() {
		return errUpdatesAvailable
	}
	return nil
}

// moduleUpdates reports version-behind modules plus the composition drift that
// `keel update` would act on, so the two commands never disagree.
func moduleUpdates(path string) ([]outdated.ModuleUpdate, []string, []string, error) {
	lk, err := lock.Read(filepath.Join(path, ".scaffold.lock"))
	if err != nil {
		// No lock → nothing to compare; not an error.
		return nil, nil, nil, nil //nolint:nilerr // a missing lock is a valid "nothing to check" state
	}
	l := module.NewFSLoader(keel.BuiltinFS)
	ups, err := outdated.ModuleUpdates(l, lk.Modules)
	if err != nil {
		return nil, nil, nil, err
	}

	rec, _, _, _, rerr := resolveUpdateRecipe(lk, path, "")
	if rerr != nil {
		// Read-only command: report what we can rather than failing.
		return ups, nil, nil, nil //nolint:nilerr // composition drift is best-effort here
	}
	// Builtin-only composite: `outdated` never fetches external sources, so a
	// recipe naming one simply yields no drift for that module.
	comp, err := module.NewComposite(keel.BuiltinFS, nil)
	if err != nil {
		return ups, nil, nil, nil //nolint:nilerr // ditto
	}
	ms, err := update.Resolve(lk, rec.ModuleNames(), compProvenance(comp), update.Options{})
	if err != nil {
		return ups, nil, nil, nil //nolint:nilerr // ditto
	}
	return ups, ms.AddedModules(), ms.OrphanedModules(), nil
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
	if rep.Empty() {
		fmt.Fprintln(out, "Everything is up to date.")
		return
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
