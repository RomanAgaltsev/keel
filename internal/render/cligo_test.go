package render_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGoCLIEmitsACobraCommandTree(t *testing.T) {
	p := planForRecipe(t, "go-cli")

	require.Contains(t, p.Files, "cmd/demo/main.go")
	require.Contains(t, p.Files, "internal/cli/root.go")
	require.Contains(t, p.Files, "internal/cli/version.go")
	require.Contains(t, p.Files, "internal/cli/root_test.go")

	require.Contains(t, p.Files["go.mod"], "github.com/spf13/cobra v1.10.2",
		"the recipe must contribute cobra to go.mod")
	require.Contains(t, p.Files["internal/cli/root.go"], "github.com/spf13/cobra")
	require.Contains(t, p.Files["cmd/demo/main.go"],
		"github.com/RomanAgaltsev/demo/internal/cli")
}

func TestGoCLIVersionCommandIsWiredToLdflags(t *testing.T) {
	// A released binary must report its tag, not "dev" -- the lesson keel's own
	// 2.2.0 learned. main injects the ldflags vars; cli renders them.
	p := planForRecipe(t, "go-cli")
	main := p.Files["cmd/demo/main.go"]
	require.Contains(t, main, `version = "dev"`)
	require.Contains(t, main, "cli.BuildInfo{Version: version, Commit: commit, Date: date}")
	require.Contains(t, p.Files["internal/cli/version.go"], "b.Version")
}
