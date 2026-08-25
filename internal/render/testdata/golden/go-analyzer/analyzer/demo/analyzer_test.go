package demo_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/RomanAgaltsev/demo/analyzer/demo"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), demo.Analyzer, "a")
}
