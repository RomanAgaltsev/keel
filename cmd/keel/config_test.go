package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/config"
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

func TestGetField(t *testing.T) {
	c := config.Config{AuthorName: "Ann", AuthorEmail: "ann@x.io", Provider: "github"}

	for key, want := range map[string]string{
		"author.name":  "Ann",
		"author.email": "ann@x.io",
		"provider":     "github",
	} {
		got, err := getField(c, key)
		require.NoError(t, err)
		require.Equal(t, want, got)
	}

	_, err := getField(c, "nope")
	require.ErrorContains(t, err, "unknown config key")
}

func TestSetField(t *testing.T) {
	var c config.Config

	require.NoError(t, setField(&c, "author.name", "Ann"))
	require.NoError(t, setField(&c, "author.email", "ann@x.io"))
	require.NoError(t, setField(&c, "provider", "github"))
	require.Equal(t, config.Config{AuthorName: "Ann", AuthorEmail: "ann@x.io", Provider: "github"}, c)

	require.ErrorContains(t, setField(&c, "nope", "x"), "unknown config key")
}
