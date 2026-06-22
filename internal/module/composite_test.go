package module_test

import (
	"sort"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/module"
)

func extFS(name, version string) fstest.MapFS {
	return fstest.MapFS{
		"module.yaml":         {Data: []byte("name: " + name + "\nversion: " + version + "\nlanguage: go\nrequires: [base-layout]\n")},
		"templates/x.go.tmpl": {Data: []byte("package x\n")},
	}
}

func TestCompositeLoadsExternalAndBuiltin(t *testing.T) {
	c, err := module.NewComposite(keel.BuiltinFS, []module.External{
		{Name: "logging", FS: extFS("logging", "1.2.0"), Source: "dir:./mods/logging", Version: "1.2.0"},
	})
	require.NoError(t, err)

	m, err := c.Load("logging")
	require.NoError(t, err)
	require.Equal(t, "logging", m.Name)
	require.Equal(t, []string{"base-layout"}, m.Requires)

	bl, err := c.Load("base-layout") // builtin still resolves
	require.NoError(t, err)
	require.Equal(t, "base-layout", bl.Name)

	tfs, err := c.TemplateFS("logging")
	require.NoError(t, err)
	_, err = tfs.Open("x.go.tmpl")
	require.NoError(t, err)

	src, ver := c.Provenance("logging")
	require.Equal(t, "dir:./mods/logging", src)
	require.Equal(t, "1.2.0", ver)
	src, _ = c.Provenance("base-layout")
	require.Equal(t, "builtin", src)
}

func TestCompositeModuleNames(t *testing.T) {
	c, err := module.NewComposite(keel.BuiltinFS, []module.External{
		{Name: "logging", FS: extFS("logging", "1.0.0"), Source: "dir:./mods/logging", Version: "1.0.0"},
	})
	require.NoError(t, err)

	names, err := c.ModuleNames()
	require.NoError(t, err)

	require.Contains(t, names, "logging")     // external is included
	require.Contains(t, names, "base-layout") // builtin is included
	require.True(t, sort.StringsAreSorted(names), "ModuleNames must be sorted")
}

func TestCompositeCollisionWithBuiltin(t *testing.T) {
	_, err := module.NewComposite(keel.BuiltinFS, []module.External{
		{Name: "lint-go", FS: extFS("lint-go", "9.9.9"), Source: "dir:./x", Version: "9.9.9"},
	})
	require.ErrorContains(t, err, "collides")
}

func TestCompositeExternalRequiresUnknownFails(t *testing.T) {
	bad := fstest.MapFS{"module.yaml": {Data: []byte("name: a\nversion: 1.0.0\nrequires: [ghost]\n")}}
	c, err := module.NewComposite(keel.BuiltinFS, []module.External{{Name: "a", FS: bad, Source: "dir:./a", Version: "1.0.0"}})
	require.NoError(t, err)
	_, err = module.Resolve(c, []string{"a"})
	require.Error(t, err) // ghost has no source
}
