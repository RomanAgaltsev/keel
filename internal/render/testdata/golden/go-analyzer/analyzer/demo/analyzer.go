// Package demo implements a demo analyzer.
package demo

import (
	"errors"
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer reports empty branch bodies -- `if cond {}`. It is a placeholder
// rule: replace it with your own. The wiring, the corpus and the test around it
// are what this scaffold exists to give you.
var Analyzer = &analysis.Analyzer{
	Name:     "demo",
	Doc:      "reports empty branch bodies",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, errors.New("inspect analyzer did not run")
	}
	insp.Preorder([]ast.Node{(*ast.IfStmt)(nil)}, func(n ast.Node) {
		stmt, ok := n.(*ast.IfStmt)
		if !ok || stmt.Body == nil || len(stmt.Body.List) > 0 {
			return
		}
		pass.Reportf(stmt.Body.Lbrace, "empty branch body")
	})
	return nil, nil
}
