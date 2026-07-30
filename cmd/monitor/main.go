package main

import (
	"fmt"
	"os"
	"time"

	"invest/internal/analysis"
	"invest/internal/cli"
	"invest/internal/market"
	"invest/pkg/data"
)

const pollInterval = 30 * time.Second

func main() {
	target := market.DefaultSecurity
	fmt.Println("正在加载K线数据计算指标...")
	klines, err := cli.FetchKLines(target.SecID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	snapshot, err := analysis.RequireLatestIndicators(klines)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	last := snapshot.LastIndex()

	// Compute T+0 targets once (indicators don't change intraday).
	sig := analysis.DefaultCompositeSignal(snapshot, data.Quote{})

	fmt.Printf("指标计算完成 (基于 %s 收盘数据)\n", klines[last].Date.Format("2006-01-02"))
	fmt.Printf("MA5: %.4f  MA20: %.4f  RSI: %.1f\n", snapshot.Current.MA5, snapshot.Current.MA20, snapshot.Current.RSI14)
	fmt.Printf("布林: %.4f ~ %.4f\n", snapshot.Current.BollLower, snapshot.Current.BollUpper)
	fmt.Printf("T+0方向: %s\n", sig.T0Dir)
	fmt.Println("─────────────────────────────────")
	fmt.Println("开始监控实时行情，等待开盘...")

	var notifiedBuy, notifiedSell bool

	for {
		if !isTradingTime() {
			time.Sleep(pollInterval)
			continue
		}

		q, err := cli.FetchQuote(target.SecID)
		if err != nil || q.Price <= 0 {
			time.Sleep(pollInterval)
			continue
		}

		// Recalculate with real-time quote.
		sig = analysis.DefaultCompositeSignal(snapshot, q)

		now := time.Now().Format("15:04:05")
		fmt.Printf("[%s] 价格: %.3f  买入位: %.3f  卖出位: %.3f\n",
			now, q.Price, sig.T0BuyPrice, sig.T0SellPrice)

		// Check buy signal
		if !notifiedBuy && sig.T0BuyPrice > 0 && q.Price <= sig.T0BuyPrice {
			msg := fmt.Sprintf("%s 触及买入位！当前 %.3f ≤ 目标 %.3f", target.Code, q.Price, sig.T0BuyPrice)
			notify("买入信号", msg)
			notifiedBuy = true
		}

		// Check sell signal
		if !notifiedSell && sig.T0SellPrice > 0 && q.Price >= sig.T0SellPrice {
			msg := fmt.Sprintf("%s 触及卖出位！当前 %.3f ≥ 目标 %.3f", target.Code, q.Price, sig.T0SellPrice)
			notify("卖出信号", msg)
			notifiedSell = true
		}

		time.Sleep(pollInterval)
	}
}
