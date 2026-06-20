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
		mods, err := moduleUpdates(path)
		if err != nil {
			return err
		}
		rep.Modules = mods
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

func moduleUpdates(path string) ([]outdated.ModuleUpdate, error) {
	lk, err := lock.Read(filepath.Join(path, ".scaffold.lock"))
	if err != nil {
		// No lock → nothing to compare; not an error.
		return nil, nil //nolint:nilerr // a missing lock is a valid "no modules to check" state
	}
	l := module.NewFSLoader(keel.BuiltinFS)
	return outdated.ModuleUpdates(l, lk.Modules)
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
		fmt.Fprintln(out, "Outdated keel modules (run keel update when available):")
		for _, u := range rep.Modules {
			fmt.Fprintf(out, "  %-32s %s -> %s\n", u.Name, u.Current, u.Latest)
		}
	}
}
