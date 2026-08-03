// Package noexit содержит анализатор, запрещающий прямой вызов os.Exit
// в функции main пакета main.
package noexit

import (
	"go/ast"
	"path/filepath"

	"golang.org/x/tools/go/analysis"
)

// Analyzer запрещает использование os.Exit в функции main пакета main.
var Analyzer = &analysis.Analyzer{
	Name: "noexit",
	Doc:  "запрещает прямой вызов os.Exit в функции main пакета main",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	// Анализируем только пакет main.
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		// Анализируем только файлы с именем main.go.
		filename := filepath.Base(pass.Fset.Position(file.Pos()).Filename)
		if filename != "main.go" {
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {
			// Ищем объявление функции main.
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "main" {
				return true
			}

			// Внутри main ищем вызовы os.Exit.
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				pkgIdent, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}

				if pkgIdent.Name == "os" && sel.Sel.Name == "Exit" {
					pass.Reportf(call.Pos(), "нельзя использовать os.Exit в функции main пакета main")
				}

				return true
			})

			return true
		})
	}

	return nil, nil
}
