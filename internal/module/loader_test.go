package module_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/module"
)

func TestLoaderLoadModule(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)

	m, err := l.Load("base-layout")
	require.NoError(t, err)
	require.Equal(t, "base-layout", m.Name)

	names, err := l.ModuleNames()
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"base-layout", "go-mod", "taskfile", "lint"}, names)

	_, err = l.Load("does-not-exist")
	require.Error(t, err)
}
