package render_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/render"
)

func baseSecAnswers() answers.Answers {
	return answers.Answers{"repo_name": "demo", "description": "a demo service", "enable_cargo_audit": true, "enable_cargo_deny": true}
}

func TestSecurityRustAll(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "security-rust"}, baseSecAnswers())
	require.NoError(t, err)
	require.Contains(t, plan.Files[".github/workflows/cargo-audit.yml"], "rustsec/audit-check@v2")
	require.Contains(t, plan.Files[".github/workflows/cargo-deny.yml"], "EmbarkStudios/cargo-deny-action@v2")
	require.Contains(t, plan.Files, "deny.toml")
	require.Contains(t, plan.Files, ".github/workflows/dependency-review.yml")
	require.Contains(t, plan.Files, ".github/workflows/actionlint.yml")
}

func TestSecurityRustGatesOff(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	a := baseSecAnswers()
	a["enable_cargo_audit"] = false
	a["enable_cargo_deny"] = false
	plan, err := render.BuildRecipe(l, []string{"base-layout", "security-rust"}, a)
	require.NoError(t, err)
	require.NotContains(t, plan.Files, ".github/workflows/cargo-audit.yml")
	require.NotContains(t, plan.Files, ".github/workflows/cargo-deny.yml")
	require.NotContains(t, plan.Files, "deny.toml")
	require.Contains(t, plan.Files, ".github/workflows/actionlint.yml") // always on
}
