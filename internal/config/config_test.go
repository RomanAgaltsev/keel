package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2/internal/config"
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

// TestPathIsUnderUserConfigDir covers the one function the 2026-08-15 review
// found wholly untested. It shapes every scaffold's defaults, so a wrong path
// silently means "no user config" rather than an error anyone would notice.
func TestPathIsUnderUserConfigDir(t *testing.T) {
	p, err := config.Path()
	require.NoError(t, err)

	base, err := os.UserConfigDir()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(base, "keel", "config.yaml"), p)
}

// TestLoadFromRejectsMalformedYAML pins the parse-error path: a corrupt config
// must be reported, never silently read as an empty one — an empty Config is
// indistinguishable from "no config", which would drop the user's defaults.
func TestLoadFromRejectsMalformedYAML(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(p, []byte("author_name: [unclosed\n"), 0o600))

	_, err := config.LoadFrom(p)
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse config")
}

// TestLoadFromUnreadableFileIsAnError separates "absent" from "present but
// unreadable" — only the first is a legitimate zero Config.
func TestLoadFromUnreadableFileIsAnError(t *testing.T) {
	dir := t.TempDir() // a directory, not a file: ReadFile fails with something other than ErrNotExist
	_, err := config.LoadFrom(dir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "read config")
}

// TestSaveToCreatesParentDirs covers SaveTo's mkdir branch, which is what makes
// `keel config set` work on a machine that has never run keel before.
func TestSaveToCreatesParentDirs(t *testing.T) {
	p := filepath.Join(t.TempDir(), "nested", "deeper", "config.yaml")

	require.NoError(t, config.SaveTo(p, config.Config{AuthorName: "Ada"}))

	got, err := config.LoadFrom(p)
	require.NoError(t, err)
	require.Equal(t, "Ada", got.AuthorName)
}
