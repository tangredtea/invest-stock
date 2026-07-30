package main

import (
	"fmt"
	"io"
	"os"

	"invest/internal/market"
	"invest/pkg/data"
	"invest/pkg/indicator"
	"invest/pkg/strategy"
)

func printHeader(klines []data.KLine, q data.Quote) {
	printHeaderTo(os.Stdout, klines, q)
}

func printHeaderTo(w io.Writer, klines []data.KLine, q data.Quote) {
	last := klines[len(klines)-1]
	target := market.DefaultSecurity
	fmt.Fprintln(w, "═══════════════════════════════════════════════════")
	fmt.Fprintf(w, "  %s %s  %s\n", target.Code, target.Name, last.Date.Format("2006-01-02"))
	fmt.Fprintln(w, "═══════════════════════════════════════════════════")
	if validQuoteForDisplay(q) {
		chg := (q.Price - q.PreClose) / q.PreClose * 100
		fmt.Fprintf(w, "  实时: %.3f  涨跌: %+.2f%%\n", q.Price, chg)
		fmt.Fprintf(w, "  今开: %.3f  最高: %.3f  最低: %.3f  昨收: %.3f\n",
			q.Open, q.High, q.Low, q.PreClose)
	} else {
		fmt.Fprintf(w, "  收盘: %.3f\n", last.Close)
	}
	fmt.Fprintln(w)
}

func validQuoteForDisplay(q data.Quote) bool {
	return q.Price > 0 && q.PreClose > 0 && q.Open > 0 && q.High >= q.Price && q.High >= q.Open && q.Low > 0 && q.Low <= q.Price && q.Low <= q.Open
}

func printIndicators(ind indicator.Indicators) {
	printIndicatorsTo(os.Stdout, ind)
}

func printIndicatorsTo(w io.Writer, ind indicator.Indicators) {
	fmt.Fprintln(w, "【技术指标】")
	fmt.Fprintf(w, "  MA5:  %.4f  MA20: %.4f  MA60: %.4f\n", ind.MA5, ind.MA20, ind.MA60)
	fmt.Fprintf(w, "  RSI(14): %.2f\n", ind.RSI14)
	fmt.Fprintf(w, "  布林带: 上轨 %.4f | 中轨 %.4f | 下轨 %.4f\n",
		ind.BollUpper, ind.BollMiddle, ind.BollLower)
	fmt.Fprintf(w, "  MACD:  DIF %.4f | DEA %.4f | 柱 %.4f\n",
		ind.MACDLine, ind.MACDSignal, ind.MACDHist)
	fmt.Fprintln(w)
}

func printSignal(sig strategy.SignalResult) {
	printSignalTo(os.Stdout, sig)
}

func printSignalTo(w io.Writer, sig strategy.SignalResult) {
	fmt.Fprintln(w, "【定投信号】")
	fmt.Fprintf(w, "  趋势: %s | 定投建议: %s\n", sig.Trend, sig.DCASignal)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "【T+0操作建议】")
	fmt.Fprintf(w, "  方向: %s\n", sig.T0Dir)
	if sig.T0Shares > 0 {
		fmt.Fprintf(w, "  建议股数: %d 股 (底仓10%%)\n", sig.T0Shares)
		if sig.T0BuyPrice > 0 {
			fmt.Fprintf(w, "  参考买入: %.3f  卖出: %.3f\n", sig.T0BuyPrice, sig.T0SellPrice)
		}
	}
	fmt.Fprintln(w, "  指标依据:")
	for _, r := range sig.T0Reasons {
		fmt.Fprintf(w, "    · %s\n", r)
	}
	fmt.Fprintf(w, "  综合: %s\n", sig.Reason)
	fmt.Fprintln(w)
}

func printBacktest(klines []data.KLine) {
	printBacktestTo(os.Stdout, klines)
}

func printBacktestTo(w io.Writer, klines []data.KLine) {
	fmt.Fprintln(w, "【回测对比(同等投入)】")
	sep := "─────────────────────────────────────────────────────────────────────────────────────────────────────────────────"
	fmt.Fprintln(w, sep)
	for _, r := range strategy.RunBacktest(klines) {
		fmt.Fprintln(w, r)
	}
	fmt.Fprintln(w, sep)
}
