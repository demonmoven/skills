package main

import (
	analysis "code.byted.org/codebase/analysis-golang-engine"
)

func main() {
	analysis.StartAnalysis(NewGoUnusedCodeAnalyzer())
}
