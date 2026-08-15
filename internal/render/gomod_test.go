package render_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2"
	"github.com/RomanAgaltsev/keel/v2/internal/answers"
	"github.com/RomanAgaltsev/keel/v2/internal/module"
	"github.com/RomanAgaltsev/keel/v2/internal/render"
)

func TestGoModEmitsCmdLayout(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"go-mod"}, answers.Answers{
		"repo_name": "demo", "description": "d", "module_path": "github.com/acme/demo", "provider": "github",
	})
	require.NoError(t, err)
	require.Contains(t, plan.Files, "cmd/demo/main.go")
	require.NotContains(t, plan.Files, "main.go") // the old root layout is gone
	require.Contains(t, plan.Files, "go.mod")
}

func TestGoModMainIsLintClean(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"go-mod"}, answers.Answers{
		"repo_name": "demo", "description": "d", "module_path": "github.com/acme/demo", "provider": "github",
	})
	require.NoError(t, err)
	main := plan.Files["cmd/demo/main.go"]

	// forbidigo (enabled in lint-go from Task 7) rejects fmt.Print* by default.
	require.NotContains(t, main, "fmt.Print")
	// The ldflags in taskfile-go inject these three; they must exist to be injectable.
	require.Contains(t, main, "version")
	require.Contains(t, main, "commit")
	require.Contains(t, main, "date")
}
