package recipe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/RomanAgaltsev/keel"
)

func TestParseRecipeBuiltinAndExternal(t *testing.T) {
	src := []byte(`
name: my-go-service
language: go
modules:
  - base-layout
  - name: logging
    source: { dir: ./mods/logging }
  - name: cache
    source:
      git: https://github.com/u/keel-mods.git
      subdir: cache
      ref: v1.2.0
`)
	var r Recipe
	require.NoError(t, yaml.Unmarshal(src, &r))
	require.Equal(t, []string{"base-layout", "logging", "cache"}, r.ModuleNames())

	require.Nil(t, r.Modules[0].Source) // builtin

	require.NotNil(t, r.Modules[1].Source)
	require.Equal(t, "./mods/logging", r.Modules[1].Source.Dir)

	require.NotNil(t, r.Modules[2].Source)
	require.Equal(t, "https://github.com/u/keel-mods.git", r.Modules[2].Source.Git)
	require.Equal(t, "cache", r.Modules[2].Source.Subdir)
	require.Equal(t, "v1.2.0", r.Modules[2].Source.Ref)
}

func TestParseRecipeRejectsBothAndNeither(t *testing.T) {
	both := []byte("modules:\n  - name: x\n    source: { dir: ./d, git: https://h/r.git }\n")
	require.Error(t, yaml.Unmarshal(both, &Recipe{}))

	neither := []byte("modules:\n  - name: x\n    source: {}\n")
	require.Error(t, yaml.Unmarshal(neither, &Recipe{}))
}

func TestParseRecipeRejectsInvalidSources(t *testing.T) {
	cases := map[string]string{
		"git without ref": "modules:\n  - name: x\n    source: { git: https://h/r.git, subdir: m }\n",
		"git ref dash":    "modules:\n  - name: x\n    source: { git: https://h/r.git, ref: \"-x\" }\n",
		"dir with subdir": "modules:\n  - name: x\n    source: { dir: ./d, subdir: m }\n",
		"dir with ref":    "modules:\n  - name: x\n    source: { dir: ./d, ref: v1 }\n",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			require.Error(t, yaml.Unmarshal([]byte(src), &Recipe{}))
		})
	}
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "r.yaml")
	require.NoError(t, os.WriteFile(p, []byte("name: x\nlanguage: go\nmodules: [base-layout]\n"), 0o644))

	r, err := LoadFile(p)
	require.NoError(t, err)
	require.Equal(t, "x", r.Name)
	require.Equal(t, []string{"base-layout"}, r.ModuleNames())

	_, err = LoadFile(filepath.Join(dir, "nope.yaml"))
	require.Error(t, err)
}

func TestLoadBuiltin(t *testing.T) {
	for _, name := range []string{"go-service", "rust-service"} {
		t.Run(name, func(t *testing.T) {
			r, err := Load(keel.BuiltinFS, name)
			require.NoError(t, err)
			require.Equal(t, name, r.Name)
			require.Contains(t, r.ModuleNames(), "base-layout")
		})
	}

	_, err := Load(keel.BuiltinFS, "does-not-exist")
	require.Error(t, err)
}
