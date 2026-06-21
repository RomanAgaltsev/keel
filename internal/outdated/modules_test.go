package outdated_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/lock"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/outdated"
)

func TestModuleUpdates(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	// "lint-go" is embedded at >= 1.0.0; pretend the repo locked an older version.
	locked := []lock.Module{
		{Name: "lint-go", Source: "builtin", Version: "0.9.0"},
		{Name: "base-layout", Source: "builtin", Version: "1.1.0"}, // up to date (assuming embedded 1.0.0)
	}
	ups, err := outdated.ModuleUpdates(l, locked)
	require.NoError(t, err)

	names := map[string]outdated.ModuleUpdate{}
	for _, u := range ups {
		names[u.Name] = u
	}
	require.Contains(t, names, "lint-go")
	require.Equal(t, "0.9.0", names["lint-go"].Current)
	require.NotContains(t, names, "base-layout") // not behind
}

func TestModuleUpdatesIgnoresUnknownAndNonBuiltin(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	locked := []lock.Module{
		{Name: "ghost", Source: "builtin", Version: "0.1.0"}, // not embedded → skipped
		{Name: "ext", Source: "git", Version: "0.1.0"},       // non-builtin → skipped
	}
	ups, err := outdated.ModuleUpdates(l, locked)
	require.NoError(t, err)
	require.Empty(t, ups)
}
