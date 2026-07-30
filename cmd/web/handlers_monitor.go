package main

import (
	"fmt"
	"time"

	"invest/internal/analysis"
	"invest/internal/auth"
	desktopnotify "invest/internal/notify"

	"golang.org/x/net/websocket"
)

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

		loaded, clientErr, err := loadAnalyzeData(ws.Request().URL.Query().Get("code"))
		if msg := analyzeWSErrorMessage(clientErr); msg != "" {
			websocket.JSON.Send(ws, map[string]string{"error": msg})
			return
		}
		if err != nil {
			websocket.JSON.Send(ws, map[string]string{"error": err.Error()})
			return
		}

		var notifiedBuy, notifiedSell bool

		for {
			q, err := fetchQuote(loaded.SecID)
			if err != nil || q.Price <= 0 {
				time.Sleep(30 * time.Second)
				continue
			}

			sig := analysis.DefaultCompositeSignal(loaded.Snapshot, q)

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
	desktopnotify.Send(title, msg)
}
