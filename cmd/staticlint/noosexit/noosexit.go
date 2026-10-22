// Package noosexit предоставляет анализатор, запрещающий использование os.Exit в функции main.
package noosexit

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer запрещает прямые вызовы os.Exit внутри функции main пакета main.
// Вместо этого рекомендуется использовать log.Fatal или возврат ошибки.
var Analyzer = &analysis.Analyzer{
	Name:     "noosexit",
	Doc:      "reports direct os.Exit calls in the main function of the main package",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	// 1. Проверяем, что это пакет main
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	// 2. Пропускаем тестовые файлы и временные директории
	// Проверяем путь пакета — если он содержит "testdata" или временные пути, пропускаем
	pkgPath := pass.Pkg.Path()
	if strings.Contains(pkgPath, "testdata") ||
	   strings.Contains(pkgPath, "go-build") ||
	   strings.Contains(pkgPath, "testmain") {
		return nil, nil
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{(*ast.FuncDecl)(nil)}

	// 3. Ищем функцию main
	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)
		if fn.Name.Name == "main" && fn.Recv == nil {
			// 4. Дополнительная проверка: пропускаем файлы из testdata
			filename := pass.Fset.Position(fn.Pos()).Filename
			if strings.Contains(filename, "testdata") ||
			   strings.Contains(filename, "go-build") {
				return
			}

			// 5. Рекурсивно обходим AST функции main в поисках вызовов
			ast.Inspect(fn, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}

				// Проверяем, является ли вызов селектором (например, os.Exit)
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					if sel.Sel.Name == "Exit" {
						if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "os" {
							// 6. Проверяем через types.Info, что это действительно пакет "os"
							if obj := pass.TypesInfo.ObjectOf(ident); obj != nil {
								if pkg, ok := obj.(*types.PkgName); ok && pkg.Imported().Path() == "os" {
									pass.Reportf(call.Pos(), "do not use os.Exit in main function, use log.Fatal or return an error instead")
								}
							}
						}
					}
				}
				return true
			})
		}
	})

	return nil, nil
}