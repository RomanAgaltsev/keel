package manifest_test

import (
	"testing"

	"github.com/RomanAgaltsev/keel/internal/manifest"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestParseManifest(t *testing.T) {
	src := []byte(`
name: ci-discipline
description: Unified CI config
version: 1.0.0
language: go
requires: [base-layout]
questions:
  - id: enable_codeql
    prompt: "Enable CodeQL?"
    type: bool
    default: true
files:
  - src: "*"
    dest: "."
    when: "{{ .enable_codeql }}"
`)
	var m manifest.Manifest
	require.NoError(t, yaml.Unmarshal(src, &m))

	require.Equal(t, "ci-discipline", m.Name)
	require.Equal(t, "go", m.Language)
	require.Equal(t, []string{"base-layout"}, m.Requires)
	require.Len(t, m.Questions, 1)
	require.Equal(t, "enable_codeql", m.Questions[0].ID)
	require.Equal(t, "bool", m.Questions[0].Type)
	require.Equal(t, true, m.Questions[0].Default)
	require.Len(t, m.Files, 1)
	require.Equal(t, "*", m.Files[0].Src)
	require.Equal(t, "{{ .enable_codeql }}", m.Files[0].When)
}
