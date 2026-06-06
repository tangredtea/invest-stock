package main

import (
	"fmt"
	"os"

	"invest/pkg/data"
	"invest/pkg/indicator"
)

func main() {
	const secid = "1.513630"
	fmt.Println("正在从东方财富获取513630数据...")
	klines, err := data.FetchKLines(secid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取K线失败: %v\n", err)
		os.Exit(1)
	}

	quote, err := data.FetchQuote(secid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取实时行情失败: %v\n", err)
	}

	closes := make([]float64, len(klines))
	for i, k := range klines {
		closes[i] = k.Close
	}
	last := len(closes) - 1
	ind := indicator.ComputeAll(closes, last)
	prevInd := indicator.ComputeAll(closes, last-1)

	printHeader(klines, quote)
	printIndicators(ind)
	printSignal(closes[last], ind, prevInd.MACDHist, quote)
	printBacktest(klines)
}
