// Команда emitcheck — анализатор «модуль эмитит чужой тип» (AD-40, AD-2):
// у каждого типа записи журнала один модуль-эмитент по contracts/events/catalog.yaml.
// Запускается как go vet -vettool=emitcheck ./internal/domain/... ./internal/application/...
// из make check.
//
// Проверяет вызовы kernel.NewReaction, kernel.NewAddressed и kernel.NewIntent
// в пакетах …/internal/{domain,application}/‹модуль›:
//   - NewReaction(m, t, …), NewAddressed(m, t, …): m — модуль своего пакета,
//     эмитент типа t по каталогу — тот же модуль;
//   - NewIntent(target, from, …): from — модуль своего пакета (намерение строит
//     функция модуля-владельца target — domain/‹target›).
//
// Каталог ищется вверх от проверяемого файла: contracts/events/catalog.yaml.
// Аргументы-переменные (не константы) пропускаются — их проверяет
// kernel.NewReaction во время работы.
package main

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.yaml.in/yaml/v3"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/unitchecker"
)

// Analyzer — проверка эмитентов типов записей.
var Analyzer = &analysis.Analyzer{
	Name: "emitcheck",
	Doc:  "AD-40: модуль эмитит только свои типы записей журнала (contracts/events/catalog.yaml)",
	Run:  run,
}

func main() { unitchecker.Main(Analyzer) }

const kernelPkg = "ant/internal/domain/kernel"

var (
	catOnce sync.Once
	catalog map[string]string // тип → эмитент
	catErr  error
)

func loadCatalog(from string) (map[string]string, error) {
	catOnce.Do(func() {
		dir := from
		for {
			p := filepath.Join(dir, "contracts", "events", "catalog.yaml")
			if b, err := os.ReadFile(p); err == nil {
				var c struct {
					Types map[string]struct {
						Emitter string `yaml:"emitter"`
					} `yaml:"types"`
				}
				if catErr = yaml.Unmarshal(b, &c); catErr != nil {
					return
				}
				catalog = map[string]string{}
				for t, d := range c.Types {
					catalog[t] = d.Emitter
				}
				return
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				catErr = fmt.Errorf("emitcheck: не найден contracts/events/catalog.yaml выше %s", from)
				return
			}
			dir = parent
		}
	})
	return catalog, catErr
}

// moduleOf — модуль пакета …/internal/{domain,application}/‹модуль›[/…].
func moduleOf(path string) string {
	path = strings.TrimSuffix(path, "_test")
	for _, layer := range []string{"/internal/domain/", "/internal/application/"} {
		if i := strings.Index(path, layer); i >= 0 {
			rest := path[i+len(layer):]
			m, _, _ := strings.Cut(rest, "/")
			return m
		}
	}
	return ""
}

func run(pass *analysis.Pass) (any, error) {
	mod := moduleOf(pass.Pkg.Path())
	if mod == "" || mod == "kernel" {
		return nil, nil
	}
	for _, f := range pass.Files {
		file := pass.Fset.File(f.Pos()).Name()
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			name := calleeName(pass, call)
			switch name {
			case "NewReaction", "NewAddressed":
				if len(call.Args) < 2 {
					return true
				}
				if m, ok := constString(pass, call.Args[0]); ok && m != mod {
					pass.Reportf(call.Pos(), "AD-40: модуль %s строит запись от имени модуля %s", mod, m)
				}
				t, ok := constString(pass, call.Args[1])
				if !ok {
					return true
				}
				cat, err := loadCatalog(filepath.Dir(file))
				if err != nil {
					pass.Reportf(call.Pos(), "%v", err)
					return true
				}
				em, known := cat[t]
				switch {
				case !known:
					pass.Reportf(call.Pos(), "AD-40: тип %s не из каталога contracts/events/catalog.yaml", t)
				case em != mod:
					pass.Reportf(call.Pos(), "AD-40: модуль %s эмитит чужой тип %s (эмитент — %s)", mod, t, em)
				}
			case "NewIntent":
				if len(call.Args) < 2 {
					return true
				}
				if from, ok := constString(pass, call.Args[1]); ok && from != mod {
					// Намерение строит функция владельца, а from — модуль, выразивший намерение:
					// в пакете владельца from — параметр, константой он бывает только у самого владельца.
					if target, ok := constString(pass, call.Args[0]); ok && target != mod {
						pass.Reportf(call.Pos(), "AD-40: намерение к %s строится не функцией модуля-владельца (%s)", target, mod)
					}
				}
			}
			return true
		})
	}
	return nil, nil
}

func calleeName(pass *analysis.Pass, call *ast.CallExpr) string {
	var id *ast.Ident
	switch fn := ast.Unparen(call.Fun).(type) {
	case *ast.SelectorExpr:
		id = fn.Sel
	case *ast.Ident:
		id = fn
	case *ast.IndexExpr:
		if sel, ok := fn.X.(*ast.SelectorExpr); ok {
			id = sel.Sel
		}
	}
	if id == nil {
		return ""
	}
	obj, ok := pass.TypesInfo.Uses[id].(*types.Func)
	if !ok || obj.Pkg() == nil || obj.Pkg().Path() != kernelPkg {
		return ""
	}
	return obj.Name()
}

func constString(pass *analysis.Pass, e ast.Expr) (string, bool) {
	tv, ok := pass.TypesInfo.Types[e]
	if !ok || tv.Value == nil || tv.Value.Kind() != constant.String {
		return "", false
	}
	return constant.StringVal(tv.Value), true
}
