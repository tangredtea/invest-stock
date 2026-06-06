package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"invest/internal/auth"
	"invest/pkg/data"
	"invest/pkg/indicator"
	"invest/pkg/strategy"

	"golang.org/x/net/websocket"
)

type analyzeResp struct {
	Code       string                  `json:"code"`
	SecID      string                  `json:"secid"`
	Klines     []data.KLine            `json:"klines"`
	Indicators indicator.Series        `json:"indicators"`
	Current    indicator.Indicators    `json:"current"`
	Signal     strategy.SignalResult   `json:"signal"`
	Backtest   []strategy.BacktestResult `json:"backtest"`
}

func handleAnalyze(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	secid := resolveSecID(code)
	if secid == "" {
		http.Error(w, `{"error":"无效股票代码"}`, 400)
		return
	}

	klines, err := data.FetchKLines(secid)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), 500)
		return
	}

	closes := make([]float64, len(klines))
	for i, k := range klines {
		closes[i] = k.Close
	}
	// Need at least 2 data points to compute the latest and prior positions.
	if len(closes) < 2 {
		http.Error(w, `{"error":"K线数量不足,无法分析"}`, 400)
		return
	}
	last := len(closes) - 1

	// Compute each indicator series exactly once, then index into it.
	series := indicator.ComputeSeries(closes)
	cur := indicatorsAt(series, last)
	prev := indicatorsAt(series, last-1)

	q, _ := data.FetchQuote(secid)
	qi := strategy.QuoteInfo{Price: q.Price, High: q.High, Low: q.Low, PreClose: q.PreClose}
	sig := strategy.Composite(closes[last], cur.MA5, cur.MA20, cur.MA60, cur.RSI14,
		cur.BollUpper, cur.BollLower, cur.MACDHist, prev.MACDHist, 100000, qi)

	bt := strategy.RunBacktest(klines)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analyzeResp{
		Code: code, SecID: secid,
		Klines: klines, Indicators: series, Current: cur,
		Signal: sig, Backtest: bt,
	})
}

// indicatorsAt extracts a single-point Indicators snapshot from an already
// computed Series by index, without recomputing any series. The values are
// bit-for-bit identical to indicator.ComputeAll(closes, i) for the same input.
func indicatorsAt(s indicator.Series, i int) indicator.Indicators {
	return indicator.Indicators{
		MA5: s.MA5[i], MA20: s.MA20[i], MA60: s.MA60[i],
		RSI14:      s.RSI14[i],
		BollUpper:  s.BollUpper[i],
		BollMiddle: s.BollMid[i],
		BollLower:  s.BollLower[i],
		MACDLine:   s.MACDLine[i],
		MACDSignal: s.MACDSignal[i],
		MACDHist:   s.MACDHist[i],
	}
}

func handleQuote(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	secid := resolveSecID(code)
	if secid == "" {
		http.Error(w, `{"error":"无效股票代码"}`, 400)
		return
	}
	q, err := data.FetchQuote(secid)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(q)
}

// makeMonitorHandler returns a WebSocket handler that authenticates the
// connection (via Sec-WebSocket-Protocol token) before streaming any data.
func makeMonitorHandler(jwtMgr *auth.JWTManager) websocket.Handler {
	return func(ws *websocket.Conn) {
		// Authenticate before streaming anything (Requirements 2.1, 2.3, 5.3, 5.4).
		token, err := extractWSToken(ws.Request())
		if err != nil {
			websocket.JSON.Send(ws, map[string]string{"error": "未认证"})
			return
		}
		if _, err := jwtMgr.Parse(token); err != nil {
			websocket.JSON.Send(ws, map[string]string{"error": "认证失败"})
			return
		}

		code := ws.Request().URL.Query().Get("code")
		secid := resolveSecID(code)
		if secid == "" {
			websocket.JSON.Send(ws, map[string]string{"error": "无效股票代码"})
			return
		}

		// Load indicators once.
		klines, err := data.FetchKLines(secid)
		if err != nil {
			websocket.JSON.Send(ws, map[string]string{"error": err.Error()})
			return
		}
		closes := make([]float64, len(klines))
		for i, k := range klines {
			closes[i] = k.Close
		}
		if len(closes) < 2 {
			websocket.JSON.Send(ws, map[string]string{"error": "K线数量不足,无法监控"})
			return
		}
		last := len(closes) - 1
		series := indicator.ComputeSeries(closes)
		cur := indicatorsAt(series, last)
		prev := indicatorsAt(series, last-1)

		var notifiedBuy, notifiedSell bool

		for {
			q, err := data.FetchQuote(secid)
			if err != nil || q.Price <= 0 {
				time.Sleep(30 * time.Second)
				continue
			}

			qi := strategy.QuoteInfo{Price: q.Price, High: q.High, Low: q.Low, PreClose: q.PreClose}
			sig := strategy.Composite(closes[last], cur.MA5, cur.MA20, cur.MA60, cur.RSI14,
				cur.BollUpper, cur.BollLower, cur.MACDHist, prev.MACDHist, 100000, qi)

			msg := map[string]any{
				"type": "tick", "time": time.Now().Format("15:04:05"),
				"price": q.Price, "signal": sig, "quote": q,
			}
			if err := websocket.JSON.Send(ws, msg); err != nil {
				return // client disconnected
			}

			// Alert on buy
			if !notifiedBuy && sig.T0BuyPrice > 0 && q.Price <= sig.T0BuyPrice {
				alert := map[string]any{
					"type": "alert", "action": "buy",
					"message": fmt.Sprintf("触及买入位 %.3f ≤ %.3f", q.Price, sig.T0BuyPrice),
					"price":   q.Price,
				}
				websocket.JSON.Send(ws, alert)
				notifyOS("买入信号", alert["message"].(string))
				notifiedBuy = true
			}

			// Alert on sell
			if !notifiedSell && sig.T0SellPrice > 0 && q.Price >= sig.T0SellPrice {
				alert := map[string]any{
					"type": "alert", "action": "sell",
					"message": fmt.Sprintf("触及卖出位 %.3f ≥ %.3f", q.Price, sig.T0SellPrice),
					"price":   q.Price,
				}
				websocket.JSON.Send(ws, alert)
				notifyOS("卖出信号", alert["message"].(string))
				notifiedSell = true
			}

			time.Sleep(30 * time.Second)
		}
	}
}

// notifyOS sends a desktop notification on macOS. On other platforms it is a
// no-op. Failures are non-fatal and never interrupt the monitor loop.
func notifyOS(title, msg string) {
	if runtime.GOOS != "darwin" {
		return
	}
	script := fmt.Sprintf(`display notification "%s" with title "%s" sound name "Glass"`, msg, title)
	_ = exec.Command("osascript", "-e", script).Run()
}
