package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/internal/lock"
)

func TestNewWithExternalDirModule(t *testing.T) {
	work := t.TempDir()

	// External module on disk next to the recipe.
	ext := filepath.Join(work, "mods", "logging")
	require.NoError(t, os.MkdirAll(filepath.Join(ext, "templates"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(ext, "module.yaml"),
		[]byte("name: logging\nversion: 1.2.0\nlanguage: go\nrequires: [base-layout]\nfiles:\n  - src: \"*\"\n    dest: \".\"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(ext, "templates", "log.go.tmpl"),
		[]byte("package log // {{ .repo_name }}\n"), 0o644))

	// Recipe file referencing builtin + the dir module.
	recPath := filepath.Join(work, "recipe.yaml")
	require.NoError(t, os.WriteFile(recPath, []byte(
		"name: ext-demo\nlanguage: go\nmodules:\n  - base-layout\n  - go-mod\n  - name: logging\n    source: { dir: ./mods/logging }\n",
	), 0o644))

	// Answers file (non-interactive).
	ansPath := filepath.Join(work, "answers.yaml")
	require.NoError(t, os.WriteFile(ansPath, []byte(
		"repo_name: demo\ndescription: d\nmodule_path: github.com/x/demo\nauthor_name: R\nauthor_email: r@x.io\nlicense: MIT\nvisibility: public\nprovider: none\ncreate_remote: false\n",
	), 0o644))

	target := filepath.Join(work, "out")
	cmd := newRootCmd()
	cmd.SetArgs([]string{"new", "--no-input", "--recipe", recPath, "--answers", ansPath, "--target", target})
	require.NoError(t, cmd.Execute())

	b, err := os.ReadFile(filepath.Join(target, "log.go"))
	require.NoError(t, err)
	require.Contains(t, string(b), "package log // demo") // rendered

	lk, err := lock.Read(filepath.Join(target, ".scaffold.lock"))
	require.NoError(t, err)
	var src string
	for _, m := range lk.Modules {
		if m.Name == "logging" {
			src = m.Source
		}
	}
	require.Equal(t, "dir:./mods/logging", src)
}
