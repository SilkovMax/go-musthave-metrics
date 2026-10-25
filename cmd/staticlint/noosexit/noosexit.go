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
var Analyzer = &analysis.Analyzer{
	Name:     "noosexit",
	Doc:      "reports direct os.Exit calls in the main function of the main package",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	// 1. Работаем только с пакетом main
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{(*ast.FuncDecl)(nil)}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)

		// 2. Ищем именно функцию main (без receiver)
		if fn.Name.Name == "main" && fn.Recv == nil {

			// 3. Получаем абсолютный путь к файлу
			filename := pass.Fset.Position(fn.Pos()).Filename

			// Игнорируем временные файлы кэша, системные папки и testdata
			if strings.Contains(filename, "go-build") ||
				strings.Contains(filename, "AppData") {
				return
			}

			// 4. Рекурсивно обходим AST функции main в поисках вызовов
			ast.Inspect(fn, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}

				// Проверяем, является ли вызов селектором (например, os.Exit)
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					if sel.Sel.Name == "Exit" {
						if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "os" {
							// 5. Убеждаемся через types.Info, что 'os' — это именно импортированный пакет
							if obj := pass.TypesInfo.ObjectOf(ident); obj != nil {
								if _, ok := obj.(*types.PkgName); ok {
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
