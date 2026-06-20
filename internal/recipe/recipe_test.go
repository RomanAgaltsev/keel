package recipe

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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
