package update_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/lock"
	"github.com/RomanAgaltsev/keel/internal/update"
)

func TestNewLockRefreshesOnlyNamedModules(t *testing.T) {
	old := lock.Lock{
		KeelVersion: "1.5.0", Recipe: "go-service",
		Modules: []lock.Module{
			{Name: "lint", Source: "builtin", Version: "1.0.0", Files: []lock.File{{Path: ".golangci.yml", SHA256: "old"}}},
			{Name: "base", Source: "builtin", Version: "1.0.0", Files: []lock.File{{Path: "README.md", SHA256: "keep"}}},
		},
		Answers: map[string]any{"repo_name": "demo"},
	}
	renderContent := map[string]string{".golangci.yml": "NEW", "README.md": "ALSO-NEW"}
	owner := map[string]string{".golangci.yml": "lint", "README.md": "base"}
	refreshed := map[string]string{"lint": "1.1.0"} // only lint bumped

	got := update.NewLock(old, renderContent, owner, refreshed, "1.6.0")

	require.Equal(t, "1.6.0", got.KeelVersion)
	require.Equal(t, "go-service", got.Recipe)
	require.Equal(t, old.Answers, got.Answers)

	byName := map[string]lock.Module{}
	for _, m := range got.Modules {
		byName[m.Name] = m
	}
	// lint refreshed: version bumped, hash = HashBytes("NEW").
	require.Equal(t, "1.1.0", byName["lint"].Version)
	require.Equal(t, lock.HashBytes([]byte("NEW")), byName["lint"].Files[0].SHA256)
	// base untouched: old entry preserved verbatim.
	require.Equal(t, "1.0.0", byName["base"].Version)
	require.Equal(t, "keep", byName["base"].Files[0].SHA256)
}
