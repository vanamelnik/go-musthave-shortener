package main

import (
	"golang.org/x/tools/go/analysis"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
)

var simpleChecks = map[string]bool{
	"S1008": true, // Simplify returning boolean expression
	"S1002": true, // Omit comparison with boolean constant
	"S1012": true, // Replace time.Now().Sub(x) with time.Since(x)
	"S1024": true, // Replace x.Sub(time.Now()) with time.Until(x)
}

func newStaticChecks() []*analysis.Analyzer {
	sc := make([]*analysis.Analyzer, 0, len(staticcheck.Analyzers)+len(simpleChecks))
	for _, a := range staticcheck.Analyzers {
		sc = append(sc, a.Analyzer) // используем все SA-анализаторы
	}
	for _, a := range simple.Analyzers {
		if simpleChecks[a.Analyzer.Name] {
			sc = append(sc, a.Analyzer)
		}
	}

	return sc
}
