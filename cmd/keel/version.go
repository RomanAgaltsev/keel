package main

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// buildInfo is the subset of runtime/debug.BuildInfo keel reports, extracted so
// the fallback logic can be tested without building a binary.
type buildInfo struct {
	Version  string // module version: "v2.1.1" for `go install`, "(devel)" for a local build
	Revision string // vcs.revision, recorded only for builds from a VCS checkout
	Time     string // vcs.time
}

// shortRevLen matches `git rev-parse --short HEAD`, which is what the Taskfile
// injects, so both build paths print a commit of the same shape.
const shortRevLen = 7

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, _ []string) error {
			v, c, d := versionInfo(version, commit, date, readBuildInfo())
			fmt.Fprintf(cmd.OutOrStdout(), "keel %s (commit %s, built %s)\n", v, c, d)
			return nil
		},
	}
}

// versionInfo decides what to report. The -ldflags values injected by the
// Taskfile and GoReleaser always win; their zero values ("dev"/"none"/"unknown")
// mean nothing was injected, and then Go's own build info fills the gap.
//
// This matters because `go install <path>@<tag>` sets no ldflags at all, so
// before this the released, correctly-installed binary reported itself as "dev"
// forever — leaving a user no way to tell which version they actually had.
func versionInfo(v, c, d string, bi buildInfo) (string, string, string) {
	// "(devel)" is what Go records for a build from a checkout; it is no more
	// informative than "dev", so it is not worth substituting.
	if v == "dev" && bi.Version != "" && bi.Version != "(devel)" {
		v = bi.Version
	}
	if c == "none" && bi.Revision != "" {
		c = bi.Revision
		if len(c) > shortRevLen {
			c = c[:shortRevLen]
		}
	}
	if d == "unknown" && bi.Time != "" {
		d = bi.Time
	}
	return v, c, d
}

// readBuildInfo adapts runtime/debug to buildInfo. It returns a zero value when
// build info is unavailable, which versionInfo treats as "keep the defaults".
func readBuildInfo() buildInfo {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return buildInfo{}
	}
	out := buildInfo{Version: bi.Main.Version}
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			out.Revision = s.Value
		case "vcs.time":
			out.Time = s.Value
		}
	}
	return out
}
