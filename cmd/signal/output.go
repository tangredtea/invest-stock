package main

import (
	"fmt"

	"invest/pkg/data"
	"invest/pkg/indicator"
	"invest/pkg/strategy"
)

func printHeader(klines []data.KLine, q data.Quote) {
	last := klines[len(klines)-1]
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Printf("  513630 港股低波红利ETF  %s\n", last.Date.Format("2006-01-02"))
	fmt.Println("═══════════════════════════════════════════════════")
	if q.Price > 0 {
		chg := (q.Price - q.PreClose) / q.PreClose * 100
		fmt.Printf("  实时: %.3f  涨跌: %+.2f%%\n", q.Price, chg)
		fmt.Printf("  今开: %.3f  最高: %.3f  最低: %.3f  昨收: %.3f\n",
			q.Open, q.High, q.Low, q.PreClose)
	} else {
		fmt.Printf("  收盘: %.3f\n", last.Close)
	}
	fmt.Println()
}

func printIndicators(ind indicator.Indicators) {
	fmt.Println("【技术指标】")
	fmt.Printf("  MA5:  %.4f  MA20: %.4f  MA60: %.4f\n", ind.MA5, ind.MA20, ind.MA60)
	fmt.Printf("  RSI(14): %.2f\n", ind.RSI14)
	fmt.Printf("  布林带: 上轨 %.4f | 中轨 %.4f | 下轨 %.4f\n",
		ind.BollUpper, ind.BollMiddle, ind.BollLower)
	fmt.Printf("  MACD:  DIF %.4f | DEA %.4f | 柱 %.4f\n",
		ind.MACDLine, ind.MACDSignal, ind.MACDHist)
	fmt.Println()
}

func printSignal(price float64, ind indicator.Indicators, prevMacdHist float64, q data.Quote) {
	holdings := 100000
	qi := strategy.QuoteInfo{
		Price: q.Price, High: q.High, Low: q.Low, PreClose: q.PreClose,
	}
	sig := strategy.Composite(
		price, ind.MA5, ind.MA20, ind.MA60, ind.RSI14,
		ind.BollUpper, ind.BollLower,
		ind.MACDHist, prevMacdHist, holdings, qi,
	)

	fmt.Println("【定投信号】")
	fmt.Printf("  趋势: %s | 定投建议: %s\n", sig.Trend, sig.DCASignal)
	fmt.Println()

	fmt.Println("【T+0操作建议】")
	fmt.Printf("  方向: %s\n", sig.T0Dir)
	if sig.T0Shares > 0 {
		fmt.Printf("  建议股数: %d 股 (底仓10%%)\n", sig.T0Shares)
		if sig.T0BuyPrice > 0 {
			fmt.Printf("  参考买入: %.3f  卖出: %.3f\n", sig.T0BuyPrice, sig.T0SellPrice)
		}
	}
	fmt.Println("  指标依据:")
	for _, r := range sig.T0Reasons {
		fmt.Printf("    · %s\n", r)
	}
	fmt.Printf("  综合: %s\n", sig.Reason)
	fmt.Println()
}

func printBacktest(klines []data.KLine) {
	fmt.Println("【回测对比(同等投入)】")
	sep := "─────────────────────────────────────────────────────────────────────────────────────────────────────────────────"
	fmt.Println(sep)
	for _, r := range strategy.RunBacktest(klines) {
		fmt.Println(r)
	}
	fmt.Println(sep)
}
