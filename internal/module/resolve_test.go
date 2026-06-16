package module_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/module"
)

func TestResolveOrdersDependenciesFirst(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	// go-mod requires base-layout, so base-layout must come first.
	order, err := module.Resolve(l, []string{"go-mod"})
	require.NoError(t, err)

	names := make([]string, len(order))
	for i, m := range order {
		names[i] = m.Name
	}
	require.Equal(t, []string{"base-layout", "go-mod"}, names)
}

func TestResolveMissingModule(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	_, err := module.Resolve(l, []string{"nope"})
	require.Error(t, err)
}
