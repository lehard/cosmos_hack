// Команда detcheck — анализатор детерминизма доменного кода (AD-4, NFR-DET-1).
// Запускается как go vet -vettool=detcheck ./internal/domain/... из make check.
//
// Запрещает в пакетах …/internal/domain/…:
//   - любые значения и типы float32, float64, complex64, complex128 и нетипизированные
//     вещественные константы (доли и уверенность — фиксированная точка *_bp,
//     измерения — целое + единица + масштаб);
//   - обход map через range: порядок обхода недетерминирован; разрешён только
//     slices.Sorted(maps.Keys(m));
//   - maps.Keys / maps.Values / maps.All вне slices.Sorted / slices.SortedFunc.
//
// Часы, случайность и пакет os ловит forbidigo (backend/.golangci.yml).
package main

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/unitchecker"
)

// Analyzer — проверка детерминизма домена.
var Analyzer = &analysis.Analyzer{
	Name: "detcheck",
	Doc:  "AD-4: в домене нет float и недетерминированного обхода map",
	Run:  run,
}

func main() { unitchecker.Main(Analyzer) }

func isDomain(path string) bool {
	path = strings.TrimSuffix(path, "_test")
	return strings.Contains(path+"/", "/internal/domain/")
}

func run(pass *analysis.Pass) (any, error) {
	if !isDomain(pass.Pkg.Path()) {
		return nil, nil
	}
	for _, f := range pass.Files {
		name := pass.Fset.File(f.Pos()).Name()
		if strings.HasSuffix(name, "_gen.go") {
			continue // сгенерированные контракты проверяет их генератор
		}
		allowedMapsCalls := map[*ast.CallExpr]bool{}
		ast.Inspect(f, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok && isFunc(pass, call, "slices", "Sorted", "SortedFunc", "SortedStableFunc") {
				for _, a := range call.Args {
					if inner, ok := ast.Unparen(a).(*ast.CallExpr); ok {
						allowedMapsCalls[inner] = true
					}
				}
			}
			return true
		})
		reported := map[ast.Node]bool{}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.RangeStmt:
				if t := pass.TypesInfo.TypeOf(n.X); t != nil {
					if _, ok := t.Underlying().(*types.Map); ok {
						pass.Reportf(n.Pos(), "AD-4: обход map через range недетерминирован — используйте slices.Sorted(maps.Keys(m))")
					}
				}
			case *ast.CallExpr:
				if isFunc(pass, n, "maps", "Keys", "Values", "All") && !allowedMapsCalls[n] {
					pass.Reportf(n.Pos(), "AD-4: maps.Keys/Values/All дают недетерминированный порядок — оберните в slices.Sorted")
				}
			case ast.Expr:
				if reported[n] {
					return true
				}
				tv, ok := pass.TypesInfo.Types[n]
				if !ok || tv.Type == nil {
					return true
				}
				if isFloat(tv.Type) {
					pass.Reportf(n.Pos(), "AD-4: float/complex в домене запрещён — фиксированная точка (*_bp) или целое + масштаб")
					// вложенные выражения того же типа не дублируем
					ast.Inspect(n, func(c ast.Node) bool {
						if c != nil {
							reported[c] = true
						}
						return true
					})
				}
			}
			return true
		})
	}
	return nil, nil
}

func isFunc(pass *analysis.Pass, call *ast.CallExpr, pkg string, names ...string) bool {
	sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok {
		// slices.Sorted[...] с явными параметрами типа
		if ix, ok := call.Fun.(*ast.IndexExpr); ok {
			sel, ok = ix.X.(*ast.SelectorExpr)
			if !ok {
				return false
			}
		} else {
			return false
		}
	}
	obj, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !ok || obj.Pkg() == nil || obj.Pkg().Path() != pkg {
		return false
	}
	for _, n := range names {
		if obj.Name() == n {
			return true
		}
	}
	return false
}

// isFloat — тип является вещественным или комплексным числом либо контейнером
// таких чисел. Поля структур не обходим: объявление поля само является
// выражением типа и будет отмечено в месте объявления.
func isFloat(t types.Type) bool {
	seen := map[types.Type]bool{}
	var walk func(types.Type) bool
	walk = func(t types.Type) bool {
		if t == nil || seen[t] {
			return false
		}
		seen[t] = true
		switch u := t.Underlying().(type) {
		case *types.Basic:
			return u.Info()&(types.IsFloat|types.IsComplex) != 0
		case *types.Pointer:
			return walk(u.Elem())
		case *types.Slice:
			return walk(u.Elem())
		case *types.Array:
			return walk(u.Elem())
		case *types.Map:
			return walk(u.Key()) || walk(u.Elem())
		case *types.Chan:
			return walk(u.Elem())
		}
		return false
	}
	return walk(t)
}
