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

func TestReleaseGoReleaserVerbatim(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	a := answers.Answers{"repo_name": "x", "description": "d", "module_path": "github.com/x/x"}
	plan, err := render.BuildRecipe(l, []string{"base-layout", "release-go"}, a)
	require.NoError(t, err)
	require.True(t, strings.Contains(plan.Files[".goreleaser.yaml"], "{{ .Version }}"))
}
