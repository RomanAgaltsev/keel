package settings_test

import (
	"go/build"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSettingsDoesNotImportProvider pins the one-way dependency: providers
// implement settings.Group, never the reverse. A cycle here would be caught by
// the compiler, but an indirect import would not be, and it would quietly make
// the engine untestable without HTTP.
func TestSettingsDoesNotImportProvider(t *testing.T) {
	pkg, err := build.Import("github.com/RomanAgaltsev/keel/v2/internal/settings", "", 0)
	require.NoError(t, err)
	for _, imp := range pkg.Imports {
		require.False(t, strings.Contains(imp, "internal/provider"),
			"internal/settings must not import %s", imp)
	}
}
