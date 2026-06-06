package main

import (
	"fmt"
	"invest/pkg/data"
	"sort"
)

func main() {
	klines, _ := data.FetchKLines("1.513630")
	// Analyze daily high-low spread (intraday volatility)
	var spreads []float64
	for _, k := range klines[60:] {
		spread := (k.High - k.Low) / k.Close * 100
		spreads = append(spreads, spread)
	}
	sort.Float64s(spreads)
	n := len(spreads)
	sum := 0.0
	for _, s := range spreads {
		sum += s
	}
	fmt.Printf("日内振幅分析 (共%d个交易日):\n", n)
	fmt.Printf("  平均: %.2f%%\n", sum/float64(n))
	fmt.Printf("  中位: %.2f%%\n", spreads[n/2])
	fmt.Printf("  25%%位: %.2f%%\n", spreads[n/4])
	fmt.Printf("  75%%位: %.2f%%\n", spreads[n*3/4])
	fmt.Printf("  最大: %.2f%%\n", spreads[n-1])

	// Count days where spread > threshold
	for _, th := range []float64{0.5, 0.8, 1.0, 1.5} {
		cnt := 0
		for _, s := range spreads {
			if s >= th {
				cnt++
			}
		}
		fmt.Printf("  振幅>=%.1f%%: %d天 (%.0f%%)\n", th, cnt, float64(cnt)/float64(n)*100)
	}

	// Simulate T+0: buy at low, sell at close (simplified)
	totalProfit := 0.0
	wins := 0
	for _, k := range klines[60:] {
		// Assume we can buy 1000 shares near daily low, sell at close
		profit := (k.Close - k.Low) * 1000
		if profit > 0 {
			wins++
		}
		totalProfit += profit
	}
	fmt.Printf("\nT+0理想模拟 (每天低点买1000股,收盘卖):\n")
	fmt.Printf("  总利润: %.0f元\n", totalProfit)
	fmt.Printf("  日均: %.1f元\n", totalProfit/float64(n))
	fmt.Printf("  胜率: %.0f%%\n", float64(wins)/float64(n)*100)
}
