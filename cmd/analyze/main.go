package main

import (
	"fmt"
	"os"

	"invest/internal/cli"
	"invest/internal/market"
)

func main() {
	klines, err := cli.FetchKLines(market.DefaultSecurity.SecID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	result, err := analyzeKLines(klines)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	writeAnalysis(os.Stdout, result)
}
