package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigSetGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	set := newConfigCmd()
	set.SetArgs([]string{"set", "author.name", "Roman Agaltsev", "--file", path})
	require.NoError(t, set.Execute())

	get := newConfigCmd()
	var out bytes.Buffer
	get.SetOut(&out)
	get.SetArgs([]string{"get", "author.name", "--file", path})
	require.NoError(t, get.Execute())
	require.Contains(t, out.String(), "Roman Agaltsev")
}
