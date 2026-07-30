package backtest

import "math"

func applyTradeStats(m *Metrics, trades []TradeRecord, initialCash float64) {
	m.TotalTrades = len(trades)
	winSum, lossSum := 0.0, 0.0
	winN, lossN, closedN := 0, 0, 0
	var turnoverSum float64
	for _, t := range trades {
		turnoverSum += t.Turnover
		if t.Action == "sell" {
			closedN++
			if t.RealizedPL > 0 {
				winSum += t.RealizedPL
				winN++
			} else {
				lossSum += t.RealizedPL
				lossN++
			}
		}
	}
	if closedN > 0 {
		m.WinRate = safe(float64(winN) / float64(closedN))
	}
	if winN > 0 {
		m.AvgWin = safe(winSum / float64(winN))
	}
	if lossN > 0 {
		m.AvgLoss = safe(lossSum / float64(lossN))
	}
	if lossN > 0 && winN > 0 && m.AvgLoss != 0 {
		m.ProfitLossRatio = safe(m.AvgWin / math.Abs(m.AvgLoss))
	}
	m.Turnover = safe(turnoverSum / initialCash)
}
