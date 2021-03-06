package main

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

var (
	// osExitAnalyzer проверяет вызов функции os.Exit() из функции main() пакета main.
	osExitAnalyzer = &analysis.Analyzer{
		Name: "osexit",
		Doc:  "check for os.Exit call in main function within package main",
		Run:  osExitRun,
	}
)

func osExitRun(pass *analysis.Pass) (interface{}, error) {
	// funcCallsList возвращает список всех функций, вызываемых из переданной функции.
	funcCallsList := func(x *ast.FuncDecl) []*ast.CallExpr {
		list := make([]*ast.CallExpr, 0, len(x.Body.List))
		for _, stmt := range x.Body.List {
			if exprStmt, ok := stmt.(*ast.ExprStmt); ok {
				if callExpr, ok := exprStmt.X.(*ast.CallExpr); ok {
					list = append(list, callExpr)
				}
			}
		}

		return list
	}

	// checkOsExitCall проверяет, является ли переданный вызов функции вызовом os.Exit().
	checkOsExitCall := func(callExpr *ast.CallExpr) bool {
		switch x := callExpr.Fun.(type) {
		case *ast.SelectorExpr:
			switch x2 := x.X.(type) {
			case *ast.Ident:
				if x2.Name == "os" && x.Sel.Name == "Exit" {
					return true
				}
			}
		}

		return false
	}

	// проверка работает только для пакета main
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}
	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			f, ok := node.(*ast.FuncDecl)
			if !ok || f.Name.Name != "main" {
				return true
			}
			for _, callExpr := range funcCallsList(f) {
				if checkOsExitCall(callExpr) {
					pass.Reportf(callExpr.Pos(), "call of os.Exit within 'main()' func of package 'main'")
				}
			}

			return true
		})
	}

	return nil, nil
}
