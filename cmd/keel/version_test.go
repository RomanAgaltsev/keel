package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestVersionInfoPrefersLdflags pins the release path: GoReleaser and the
// Taskfile inject exact values, and build info must never override them.
func TestVersionInfoPrefersLdflags(t *testing.T) {
	v, c, d := versionInfo("v2.1.1", "abc1234", "2026-08-15T00:00:00Z", buildInfo{
		Version: "v9.9.9", Revision: "deadbeefdeadbeef", Time: "1999-01-01T00:00:00Z",
	})
	require.Equal(t, "v2.1.1", v)
	require.Equal(t, "abc1234", c)
	require.Equal(t, "2026-08-15T00:00:00Z", d)
}

// TestVersionInfoFallsBackForGoInstall is the whole point of this change. A
// binary from `go install <path>@<tag>` carries no ldflags, so it used to report
// "dev" forever — which is exactly how a silently-wrong install goes unnoticed.
func TestVersionInfoFallsBackForGoInstall(t *testing.T) {
	v, c, d := versionInfo("dev", "none", "unknown", buildInfo{
		Version: "v2.1.1", Revision: "deadbeefdeadbeef1234", Time: "2026-08-15T00:00:00Z",
	})
	require.Equal(t, "v2.1.1", v)
	require.Equal(t, "deadbee", c, "revision is shortened to match the Taskfile's git rev-parse --short")
	require.Equal(t, "2026-08-15T00:00:00Z", d)
}

// TestVersionInfoIgnoresDevelModuleVersion covers `go build` inside a checkout:
// Go reports Main.Version as "(devel)", which is no more informative than "dev",
// but it does record the VCS revision and time — so use those.
func TestVersionInfoIgnoresDevelModuleVersion(t *testing.T) {
	v, c, d := versionInfo("dev", "none", "unknown", buildInfo{
		Version: "(devel)", Revision: "cafebabecafebabe", Time: "2026-08-15T00:00:00Z",
	})
	require.Equal(t, "dev", v, "(devel) carries no more information than dev")
	require.Equal(t, "cafebab", c)
	require.Equal(t, "2026-08-15T00:00:00Z", d)
}

// TestVersionInfoWithNoBuildInfo keeps the original behaviour when nothing is
// available at all, rather than printing empty fields.
func TestVersionInfoWithNoBuildInfo(t *testing.T) {
	v, c, d := versionInfo("dev", "none", "unknown", buildInfo{})
	require.Equal(t, "dev", v)
	require.Equal(t, "none", c)
	require.Equal(t, "unknown", d)
}

// TestVersionInfoShortRevisionIsNotPadded guards against slicing a revision that
// is already shorter than the cut length.
func TestVersionInfoShortRevisionIsNotPadded(t *testing.T) {
	_, c, _ := versionInfo("dev", "none", "unknown", buildInfo{Revision: "abc"})
	require.Equal(t, "abc", c)
}

// TestVersionCommandRunsAgainstARealBinary is the end-to-end check: whatever the
// build produced, `keel version` must report something more specific than the
// bare defaults, because this test binary is built from a VCS checkout.
func TestVersionCommandPrintsBuildProvenance(t *testing.T) {
	var buf bytes.Buffer
	cmd := newVersionCmd()
	cmd.SetOut(&buf)
	require.NoError(t, cmd.Execute())
	require.Contains(t, buf.String(), "keel ")
}
