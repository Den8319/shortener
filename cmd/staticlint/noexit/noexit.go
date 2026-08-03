// Package noexit содержит анализатор, запрещающий прямой вызов os.Exit
// в функции main пакета main.
package noexit

import (
	"go/ast"
	"os"
	"path/filepath"
	"strings"

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

	// Корневая директория проекта — текущая рабочая директория.
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	for _, file := range pass.Files {
		// Анализируем только файлы, находящиеся внутри директории проекта.
		filename := pass.Fset.Position(file.Pos()).Filename
		if !isInDir(filename, wd) {
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

// isInDir проверяет, что файл path находится внутри директории dir.
func isInDir(path, dir string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(dir, abs)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
