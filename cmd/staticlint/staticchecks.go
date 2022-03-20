package main

import (
	"strings"

	"golang.org/x/tools/go/analysis"
	"honnef.co/go/tools/staticcheck"
)

var alternativeChecks = map[string]bool{
	"S1008": true, // Simplify returning boolean expression
	"S1024": true, // Replace x.Sub(time.Now()) with time.Until(x)
}

func newStaticChecks() []*analysis.Analyzer {
	sc := make([]*analysis.Analyzer, 0)
	for _, v := range staticcheck.Analyzers {
		name := v.Analyzer.Name
		if strings.HasPrefix(name, "SA") || alternativeChecks[name] {
			sc = append(sc, v.Analyzer)
		}
	}

	return sc
}
