package render_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2"
	"github.com/RomanAgaltsev/keel/v2/internal/answers"
	"github.com/RomanAgaltsev/keel/v2/internal/module"
	"github.com/RomanAgaltsev/keel/v2/internal/render"
)

// CONTRIBUTING.md is per-language: the Go guide names gofumpt, depguard and
// govulncheck, none of which exist in a Rust repo. governance stays language:any
// and carries only the files that are the same either way.
func contributingAnswers() answers.Answers {
	return answers.Answers{
		"repo_name": "demo", "description": "d", "module_path": "github.com/acme/demo",
		"author_name": "Ada Lovelace", "author_email": "ada@example.com",
	}
}

func TestContributingGo(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"contributing-go"}, contributingAnswers())
	require.NoError(t, err)
	require.Contains(t, plan.Files, "CONTRIBUTING.md")
	got := plan.Files["CONTRIBUTING.md"]
	require.Contains(t, got, "`demo` is a Go module at `github.com/acme/demo`")
	require.Contains(t, got, "gofumpt")
	require.Contains(t, got, "depguard")
	// The required-check list must name the job names the Go recipe's workflows
	// actually define; a required check that never reports blocks every PR.
	// security-go emits one security.yml whose jobs are codeql / govulncheck /
	// dependency-review / actionlint.
	require.Contains(t, got, "`dependency-review`")
	require.Contains(t, got, "`codeql` and\n  `govulncheck` when those are enabled")
	require.NotContains(t, got, "cargo")
}

func TestContributingRust(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"contributing-rust"}, contributingAnswers())
	require.NoError(t, err)
	require.Contains(t, plan.Files, "CONTRIBUTING.md")
	got := plan.Files["CONTRIBUTING.md"]
	require.Contains(t, got, "`demo` is a Rust crate")
	require.Contains(t, got, "cargo nextest")
	require.Contains(t, got, "`audit`, `deny` and `coverage`")
	require.NotContains(t, got, "gofumpt")
	require.NotContains(t, got, "govulncheck")
}

func TestContributingModulesDeclareTheirLanguage(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	// scaffold.checkLanguages rejects a module whose language conflicts with the
	// recipe's, which is what keeps the Go guide out of a Rust repo.
	for name, want := range map[string]string{
		"contributing-go":   "go",
		"contributing-rust": "rust",
		"governance":        "any",
	} {
		m, err := l.Load(name)
		require.NoError(t, err)
		require.Equal(t, want, m.Language, "module %s", name)
	}
}
