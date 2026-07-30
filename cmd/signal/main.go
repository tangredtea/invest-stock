package main

import (
	"fmt"
	"os"

	"invest/internal/analysis"
	"invest/internal/cli"
	"invest/internal/market"
)

func main() {
	target := market.DefaultSecurity
	fmt.Printf("正在从东方财富获取%s数据...\n", target.Code)
	klines, err := cli.FetchKLines(target.SecID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	quote, err := cli.FetchQuote(target.SecID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	snapshot, err := analysis.RequireLatestIndicators(klines)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	printHeader(klines, quote)
	printIndicators(snapshot.Current)
	printSignal(analysis.DefaultCompositeSignal(snapshot, quote))
	printBacktest(klines)
}
