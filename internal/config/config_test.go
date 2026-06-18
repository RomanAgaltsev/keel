package config_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/config"
)

func TestLoadMissingReturnsZero(t *testing.T) {
	c, err := config.LoadFrom(filepath.Join(t.TempDir(), "nope.yaml"))
	require.NoError(t, err)
	require.Equal(t, config.Config{}, c)
}

func TestSaveThenLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	want := config.Config{AuthorName: "Roman Agaltsev", AuthorEmail: "roman-agalcev@yandex.ru", Provider: "github"}
	require.NoError(t, config.SaveTo(path, want))

	got, err := config.LoadFrom(path)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
