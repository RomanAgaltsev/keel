package render_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGoAnalyzerEmitsAnAnalysisSkeleton(t *testing.T) {
	p := planForRecipe(t, "go-analyzer")

	// package_name sanitises repo_name; "demo" is already legal, so the
	// directory is analyzer/demo/.
	require.Contains(t, p.Files, "analyzer/demo/analyzer.go")
	require.Contains(t, p.Files, "analyzer/demo/analyzer_test.go")
	require.Contains(t, p.Files, "analyzer/demo/testdata/src/a/a.go")
	require.Contains(t, p.Files, "cmd/demo/main.go")

	require.Contains(t, p.Files["go.mod"], "golang.org/x/tools v0.49.0",
		"the recipe must contribute x/tools to go.mod")
	require.Contains(t, p.Files["cmd/demo/main.go"], "singlechecker.Main")
	require.Contains(t, p.Files["analyzer/demo/analyzer.go"], "analysis.Analyzer")
}

func TestGoAnalyzerCorpusCarriesAWantDirective(t *testing.T) {
	// analysistest matches diagnostics by line against // want comments. A
	// corpus without one would make the emitted test vacuous.
	p := planForRecipe(t, "go-analyzer")
	require.Contains(t, p.Files["analyzer/demo/testdata/src/a/a.go"], `// want "empty branch body"`)
	require.Contains(t, p.Files["analyzer/demo/analyzer_test.go"], "analysistest.Run")
}

func TestGoAnalyzerImportsWhatItDeclares(t *testing.T) {
	// `go mod tidy` runs first in the scaffold's own `task ci` and strips any
	// declared dependency nothing imports -- recorded in the 2.7.0 execution
	// notes. The emitted analyzer must genuinely import x/tools.
	p := planForRecipe(t, "go-analyzer")
	require.Contains(t, p.Files["analyzer/demo/analyzer.go"], `"golang.org/x/tools/go/analysis"`)
}
