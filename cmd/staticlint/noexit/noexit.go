// Package noexit содержит анализатор, запрещающий прямой вызов os.Exit
// в функции main пакета main.
package noexit

import (
	"go/ast"
	"go/types"

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

				// Для квалифицированных идентификаторов (pkg.Func) объект
				// функции находится в карте Uses для имени функции (sel.Sel).
				obj, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
				if !ok {
					return true
				}

				// Проверяем канонический путь пакета и имя функции.
				if obj.Pkg() != nil && obj.Pkg().Path() == "os" && obj.Name() == "Exit" {
					pass.Reportf(call.Pos(), "нельзя использовать os.Exit в функции main пакета main")
				}

				return true
			})

			return true
		})
	}

	return nil, nil
}
