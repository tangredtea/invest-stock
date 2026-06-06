package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"invest/pkg/data"
	"invest/pkg/indicator"
	"invest/pkg/strategy"
)

const pollInterval = 30 * time.Second

func main() {
	const secid = "1.513630"
	fmt.Println("正在加载K线数据计算指标...")
	klines, err := data.FetchKLines(secid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取K线失败: %v\n", err)
		os.Exit(1)
	}

	closes := make([]float64, len(klines))
	for i, k := range klines {
		closes[i] = k.Close
	}
	last := len(closes) - 1
	ind := indicator.ComputeAll(closes, last)
	prevInd := indicator.ComputeAll(closes, last-1)

	// Compute T+0 targets once (indicators don't change intraday)
	qi := strategy.QuoteInfo{} // will be filled per tick
	sig := strategy.Composite(
		closes[last], ind.MA5, ind.MA20, ind.MA60, ind.RSI14,
		ind.BollUpper, ind.BollLower,
		ind.MACDHist, prevInd.MACDHist, 100000, qi,
	)

	fmt.Printf("指标计算完成 (基于 %s 收盘数据)\n", klines[last].Date.Format("2006-01-02"))
	fmt.Printf("MA5: %.4f  MA20: %.4f  RSI: %.1f\n", ind.MA5, ind.MA20, ind.RSI14)
	fmt.Printf("布林: %.4f ~ %.4f\n", ind.BollLower, ind.BollUpper)
	fmt.Printf("T+0方向: %s\n", sig.T0Dir)
	fmt.Println("─────────────────────────────────")
	fmt.Println("开始监控实时行情，等待开盘...")

	var notifiedBuy, notifiedSell bool

	for {
		if !isTradingTime() {
			time.Sleep(pollInterval)
			continue
		}

		q, err := data.FetchQuote(secid)
		if err != nil || q.Price <= 0 {
			time.Sleep(pollInterval)
			continue
		}

		// Recalculate with real-time quote
		qi = strategy.QuoteInfo{
			Price: q.Price, High: q.High, Low: q.Low, PreClose: q.PreClose,
		}
		sig = strategy.Composite(
			closes[last], ind.MA5, ind.MA20, ind.MA60, ind.RSI14,
			ind.BollUpper, ind.BollLower,
			ind.MACDHist, prevInd.MACDHist, 100000, qi,
		)

		now := time.Now().Format("15:04:05")
		fmt.Printf("[%s] 价格: %.3f  买入位: %.3f  卖出位: %.3f\n",
			now, q.Price, sig.T0BuyPrice, sig.T0SellPrice)

		// Check buy signal
		if !notifiedBuy && sig.T0BuyPrice > 0 && q.Price <= sig.T0BuyPrice {
			msg := fmt.Sprintf("513630 触及买入位！当前 %.3f ≤ 目标 %.3f", q.Price, sig.T0BuyPrice)
			notify("买入信号", msg)
			notifiedBuy = true
		}

		// Check sell signal
		if !notifiedSell && sig.T0SellPrice > 0 && q.Price >= sig.T0SellPrice {
			msg := fmt.Sprintf("513630 触及卖出位！当前 %.3f ≥ 目标 %.3f", q.Price, sig.T0SellPrice)
			notify("卖出信号", msg)
			notifiedSell = true
		}

		time.Sleep(pollInterval)
	}
}

func isTradingTime() bool {
	now := time.Now()
	h, m := now.Hour(), now.Minute()
	t := h*60 + m
	// A-share trading: 9:30-11:30, 13:00-15:00
	return (t >= 9*60+30 && t <= 11*60+30) || (t >= 13*60 && t <= 15*60)
}

func notify(title, msg string) {
	// Terminal beep
	fmt.Print("\a")
	fmt.Printf("\n🔔 %s: %s\n\n", title, msg)

	// macOS notification center
	script := fmt.Sprintf(`display notification "%s" with title "%s" sound name "Glass"`, msg, title)
	exec.Command("osascript", "-e", script).Run()
}
