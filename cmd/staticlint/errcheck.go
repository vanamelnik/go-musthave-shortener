package main

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

var (
	ErrcheckAnalyzer = &analysis.Analyzer{
		Name: "errcheck",
		Doc:  "check for unchecked errors",
		Run:  run,
	}
	errorType = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
)

func isErrorType(t types.Type) bool {
	return types.Implements(t, errorType)
}

func run(pass *analysis.Pass) (interface{}, error) {
	expr := func(x *ast.ExprStmt) {
		if call, ok := x.X.(*ast.CallExpr); ok {
			if isReturnError(pass, call) {
				pass.Reportf(x.Pos(), "expression returns unchecked error")
			}
		}
	}
	tupleFunc := func(x *ast.AssignStmt) {
		if call, ok := x.Rhs[0].(*ast.CallExpr); ok {
			results := resultErrors(pass, call)
			for i := 0; i < len(x.Lhs); i++ {
				if id, ok := x.Lhs[i].(*ast.Ident); ok && id.Name == "_" && results[i] {
					pass.Reportf(id.NamePos, "assignment with unchecked error")
				}
			}
		}
	}
	errfunc := func(x *ast.AssignStmt) {
		for i := 0; i < len(x.Lhs); i++ {
			if id, ok := x.Lhs[i].(*ast.Ident); ok {
				// вызов функции справа
				if call, ok := x.Rhs[i].(*ast.CallExpr); ok {
					if id.Name == "_" && isReturnError(pass, call) {
						pass.Reportf(id.NamePos, "assignment with unchecked error")
					}
				}
			}
		}
	}
	gofunc := func(x *ast.GoStmt) {
		if isReturnError(pass, x.Call) {
			pass.Reportf(x.Pos(), "go-call of function returning an error")
		}
	}
	deferfunc := func(x *ast.DeferStmt) {
		if isReturnError(pass, x.Call) {
			pass.Reportf(x.Pos(), "defer function returning an error")
		}
	}

	for _, file := range pass.Files {
		// функцией ast.Inspect проходим по всем узлам AST
		ast.Inspect(file, func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.ExprStmt: // выражение
				expr(x)
			case *ast.AssignStmt: // опрератор присваивания
				// справа одно выражение x,y := myfunc()
				if len(x.Rhs) == 1 {
					tupleFunc(x)
				} else {
					// справа несколько выражений x,y := z,myfunc()
					errfunc(x)
				}
			case *ast.GoStmt:
				gofunc(x)
			case *ast.DeferStmt:
				deferfunc(x)
			}
			return true
		})
	}
	return nil, nil
}

func resultErrors(pass *analysis.Pass, call *ast.CallExpr) []bool {
	switch t := pass.TypesInfo.Types[call].Type.(type) {
	case *types.Named:
		return []bool{isErrorType(t)}
	case *types.Pointer:
		return []bool{isErrorType(t)}
	case *types.Tuple:
		s := make([]bool, t.Len())
		for i := 0; i < t.Len(); i++ {
			switch mt := t.At(i).Type().(type) {
			case *types.Named:
				s[i] = isErrorType(mt)
			case *types.Pointer:
				s[i] = isErrorType(mt)
			}
		}
		return s
	}
	return []bool{false}
}

func isReturnError(pass *analysis.Pass, call *ast.CallExpr) bool {
	for _, isError := range resultErrors(pass, call) {
		if isError {
			return true
		}
	}
	return false
}
