package render_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/answers"
	"github.com/RomanAgaltsev/keel/internal/module"
	"github.com/RomanAgaltsev/keel/internal/render"
)

func TestCargoModRenders(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "cargo-mod"}, answers.Answers{
		"repo_name":    "demo",
		"description":  "a demo service",
		"author_name":  "Roman Agaltsev",
		"author_email": "roman-agalcev@yandex.ru",
		"license":      "MIT",
	})
	require.NoError(t, err)

	cargo := plan.Files["Cargo.toml"]
	require.Contains(t, cargo, `name = "demo"`)
	require.Contains(t, cargo, `edition = "2024"`)
	require.Contains(t, cargo, `license = "MIT"`)
	require.Contains(t, cargo, "Roman Agaltsev <roman-agalcev@yandex.ru>")

	main := plan.Files["src/main.rs"]
	require.Contains(t, main, "fn main()")
	require.Contains(t, main, "demo")
}
