package recipe

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestParseRecipe(t *testing.T) {
	src := []byte(`
name: go-service
language: go
modules: [base-layout, go-mod]
`)
	var r Recipe
	require.NoError(t, yaml.Unmarshal(src, &r))
	require.Equal(t, "go-service", r.Name)
	require.Equal(t, "go", r.Language)
	require.Equal(t, []string{"base-layout", "go-mod"}, r.Modules)
}
