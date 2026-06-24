package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// examplesDir is the repo-root examples/ tree, relative to this package (cmd/keel).
// `go test` runs with the working directory set to the package dir.
const examplesDir = "../../examples"

// TestExamplesDryRun runs every shipped example through `keel new --no-input
// --dry-run` so the examples cannot silently rot as modules/recipes evolve.
// --dry-run touches neither disk nor network, so no provider tokens are required
// even for examples whose answers set create_remote: true.
func TestExamplesDryRun(t *testing.T) {
	cases := []struct {
		name    string
		recipe  string // builtin name or a recipe-file path
		answers string
	}{
		{"go-service", "go-service", filepath.Join(examplesDir, "answers", "go-service.yaml")},
		{"rust-service", "rust-service", filepath.Join(examplesDir, "answers", "rust-service.yaml")},
		{"local-only", "go-service", filepath.Join(examplesDir, "answers", "local-only.yaml")},
		{"ci", "go-service", filepath.Join(examplesDir, "answers", "ci.yaml")},
		{"gitlab", "go-service", filepath.Join(examplesDir, "answers", "gitlab.yaml")},
		{"custom-recipe", filepath.Join(examplesDir, "custom-recipe", "recipe.yaml"), filepath.Join(examplesDir, "answers", "local-only.yaml")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newRootCmd()
			cmd.SetArgs([]string{
				"new", "--no-input", "--dry-run",
				"--recipe", tc.recipe,
				"--answers", tc.answers,
				"--target", t.TempDir(),
			})
			require.NoError(t, cmd.Execute())
		})
	}
}
