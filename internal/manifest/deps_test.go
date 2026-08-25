package manifest_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2/internal/manifest"
)

func TestValidateDepsAcceptsAWellFormedBlock(t *testing.T) {
	deps := []manifest.Dep{
		{Path: "golang.org/x/tools", Version: "v0.38.0"},
		{Path: "github.com/spf13/cobra", Version: "v1.10.2"},
	}
	require.NoError(t, manifest.ValidateDeps(deps, "go"))
}

func TestValidateDepsAcceptsAnEmptyBlock(t *testing.T) {
	// Twenty-one of keel's modules emit no Go code at all. An absent block is
	// the common case and must not require `language: go`.
	require.NoError(t, manifest.ValidateDeps(nil, "any"))
}

func TestValidateDepsRejectsMalformedEntries(t *testing.T) {
	for _, tc := range []struct {
		name string
		deps []manifest.Dep
		want string
	}{
		{
			name: "empty path",
			deps: []manifest.Dep{{Path: "  ", Version: "v1.0.0"}},
			want: "empty path",
		},
		{
			name: "version in the path",
			deps: []manifest.Dep{{Path: "golang.org/x/tools@v0.38.0", Version: "v0.38.0"}},
			want: "not in the path",
		},
		{
			name: "empty version",
			deps: []manifest.Dep{{Path: "golang.org/x/tools", Version: ""}},
			want: "empty version",
		},
		{
			name: "version without the v prefix",
			deps: []manifest.Dep{{Path: "golang.org/x/tools", Version: "0.38.0"}},
			want: "v-prefixed semver",
		},
		{
			name: "duplicate path",
			deps: []manifest.Dep{
				{Path: "golang.org/x/tools", Version: "v0.38.0"},
				{Path: "golang.org/x/tools", Version: "v0.39.0"},
			},
			want: "duplicate path",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := manifest.ValidateDeps(tc.deps, "go")
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestValidateDepsRejectsANonGoModule(t *testing.T) {
	// No Rust consumer exists, so the cargo path is unbuilt (spec §8). A block
	// on a non-Go module is a hard error, never a silent no-op: a silently
	// dropped declaration is the failure the emits Needs vocabulary was closed
	// to prevent.
	deps := []manifest.Dep{{Path: "serde", Version: "v1.0.0"}}
	err := manifest.ValidateDeps(deps, "rust")
	require.Error(t, err)
	require.Contains(t, err.Error(), "only go modules")
}

func TestManifestValidateCoversBothContracts(t *testing.T) {
	m := manifest.Manifest{
		Name:     "bad",
		Language: "go",
		Emits:    manifest.Emits{Needs: []string{"not_a_capability"}},
	}
	require.Error(t, m.Validate(), "a bad emits block must fail Manifest.Validate")

	m = manifest.Manifest{
		Name:     "bad",
		Language: "go",
		Deps:     []manifest.Dep{{Path: "x", Version: "nope"}},
	}
	require.Error(t, m.Validate(), "a bad deps block must fail Manifest.Validate")
}
