package main

import (
	"fmt"
	"io"
)

func writeAnalysis(w io.Writer, result analysisResult) {
	fmt.Fprintf(w, "日内振幅分析 (共%d个交易日):\n", result.Count)
	fmt.Fprintf(w, "  平均: %.2f%%\n", result.Average)
	fmt.Fprintf(w, "  中位: %.2f%%\n", result.Median)
	fmt.Fprintf(w, "  25%%位: %.2f%%\n", result.P25)
	fmt.Fprintf(w, "  75%%位: %.2f%%\n", result.P75)
	fmt.Fprintf(w, "  最大: %.2f%%\n", result.Max)
	for _, stat := range result.Thresholds {
		fmt.Fprintf(w, "  振幅>=%.1f%%: %d天 (%.0f%%)\n", stat.Threshold, stat.Count, stat.Percent)
	}

	fmt.Fprintln(w, "\nT+0理想模拟 (每天低点买1000股,收盘卖):")
	fmt.Fprintf(w, "  总利润: %.0f元\n", result.TotalProfit)
	fmt.Fprintf(w, "  日均: %.1f元\n", result.DailyProfit)
	fmt.Fprintf(w, "  胜率: %.0f%%\n", result.WinRate)
}
