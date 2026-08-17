package render_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2"
	"github.com/RomanAgaltsev/keel/v2/internal/answers"
	"github.com/RomanAgaltsev/keel/v2/internal/module"
	"github.com/RomanAgaltsev/keel/v2/internal/render"
)

func TestReleaseRustRenders(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"base-layout", "release-rust"}, answers.Answers{"repo_name": "demo", "description": "a demo service", "module_path": "github.com/acme/demo", "provider": "github"})
	require.NoError(t, err)
	require.Contains(t, plan.Files[".github/workflows/release-plz.yml"], "MarcoIeni/release-plz-action")
	require.Contains(t, plan.Files, "release-plz.toml")
}

// TestReleaseRustStartsAtZeroOne answers the half of #55 that was flagged as
// unchecked rather than known-broken. release-plz takes the version from
// Cargo.toml rather than from a manifest file of its own, and cargo-mod
// scaffolds 0.1.0 -- Cargo's own convention. So Rust never had the Go defect,
// and this pins that rather than changing anything.
func TestReleaseRustStartsAtZeroOne(t *testing.T) {
	cargo := planForRecipe(t, "rust-service").Files["Cargo.toml"]
	require.Contains(t, cargo, `version = "0.1.0"`)
}
