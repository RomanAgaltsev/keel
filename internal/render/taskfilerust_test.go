package render_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/render"
)

func TestTaskfileRustRenders(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "taskfile-rust"}, answers.Answers{
		"repo_name": "demo",
	})
	require.NoError(t, err)

	tf := plan.Files["Taskfile.yml"]
	require.Contains(t, tf, "cargo nextest run")
	require.Contains(t, tf, "cargo clippy")
	// Verbatim: Task's own template vars must survive keel's renderer untouched.
	require.True(t, strings.Contains(tf, "{{.ROOT_DIR}}"), "Task vars must be preserved verbatim")
	require.Contains(t, tf, "cargo install --root . --version")
}
