package render_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2"
	"github.com/RomanAgaltsev/keel/v2/internal/answers"
	"github.com/RomanAgaltsev/keel/v2/internal/module"
	"github.com/RomanAgaltsev/keel/v2/internal/render"
)

func taskfileRust(t *testing.T) string {
	t.Helper()
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "taskfile-rust"}, answers.Answers{
		"repo_name":   "demo",
		"description": "a demo service",
		"module_path": "github.com/acme/demo",
		"provider":    "github",
	})
	require.NoError(t, err)
	return plan.Files["Taskfile.yml"]
}

func TestTaskfileRustRenders(t *testing.T) {
	tf := taskfileRust(t)
	require.Contains(t, tf, "cargo nextest run")
	require.Contains(t, tf, "cargo clippy")
	// Verbatim: Task's own template vars must survive keel's renderer untouched.
	require.True(t, strings.Contains(tf, "{{.ROOT_DIR}}"), "Task vars must be preserved verbatim")
	require.Contains(t, tf, "cargo install --root . --version")
}

func TestTaskfileRustExposesKeelTasks(t *testing.T) {
	got := taskfileRust(t)
	require.Contains(t, got, "keel:outdated:")
	require.Contains(t, got, "keel:update:")
	require.NotContains(t, taskBlock(t, got, "ci"), "keel")
}
