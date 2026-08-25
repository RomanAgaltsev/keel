package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2/internal/manifest"
)

func TestStaleReportsBehindPins(t *testing.T) {
	got := stale(
		map[string][]manifest.Dep{
			"go-analyzer": {{Path: "golang.org/x/tools", Version: "v0.38.0"}},
			"go-cli":      {{Path: "github.com/spf13/cobra", Version: "v1.10.2"}},
		},
		map[string]string{
			"golang.org/x/tools":     "v0.39.0", // behind
			"github.com/spf13/cobra": "v1.10.2", // current
		},
	)
	require.Len(t, got, 1)
	require.Contains(t, got[0], "go-analyzer")
	require.Contains(t, got[0], "golang.org/x/tools")
	require.Contains(t, got[0], "v0.39.0")
}

func TestStaleIsQuietWhenEverythingIsCurrent(t *testing.T) {
	got := stale(
		map[string][]manifest.Dep{"go-cli": {{Path: "github.com/spf13/cobra", Version: "v1.10.2"}}},
		map[string]string{"github.com/spf13/cobra": "v1.10.2"},
	)
	require.Empty(t, got)
}

func TestStaleIgnoresAPinAheadOfTheProxy(t *testing.T) {
	// A pin can lead the proxy briefly after a release. That is not drift.
	got := stale(
		map[string][]manifest.Dep{"m": {{Path: "x", Version: "v1.2.0"}}},
		map[string]string{"x": "v1.1.0"},
	)
	require.Empty(t, got)
}
