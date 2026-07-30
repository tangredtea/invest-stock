package strategy

import "fmt"

// Composite generates DCA signal + T+0 guidance.
func Composite(price, ma5, ma20, ma60, rsi float64,
	bollUpper, bollLower, macdHist, prevMacdHist float64,
	holdings int, quote QuoteInfo) SignalResult {

	trend := DetectTrend(ma5, ma20, ma60)
	reasons := ""

	// === DCA signal ===
	dcaScore := 0
	switch trend {
	case TrendUp:
		dcaScore++
		reasons += "趋势向上; "
	case TrendDown:
		dcaScore--
		reasons += "趋势向下; "
	}
	if macdHist > 0 && prevMacdHist <= 0 {
		dcaScore++
		reasons += "MACD金叉; "
	} else if macdHist < 0 && prevMacdHist >= 0 {
		dcaScore--
		reasons += "MACD死叉; "
	}

	var dcaSig Signal
	switch {
	case dcaScore >= 2:
		dcaSig = StrongBuy
	case dcaScore >= 1:
		dcaSig = Buy
	case dcaScore <= -2:
		dcaSig = StrongSell
	case dcaScore <= -1:
		dcaSig = Sell
	default:
		dcaSig = Hold
	}

	// === T+0 direction ===
	t0Score := 0
	var t0Details []string
	if ma5 > ma20 {
		t0Score++
		t0Details = append(t0Details, "MA5>MA20偏多")
	} else {
		t0Score--
		t0Details = append(t0Details, "MA5<MA20偏空")
	}
	if macdHist > prevMacdHist {
		t0Score++
		t0Details = append(t0Details, "MACD柱放大")
	} else {
		t0Score--
		t0Details = append(t0Details, "MACD柱缩小")
	}
	if rsi > 0 && rsi < 45 {
		t0Score++
		t0Details = append(t0Details, fmt.Sprintf("RSI%.1f偏低有反弹空间", rsi))
	} else if rsi > 60 {
		t0Score--
		t0Details = append(t0Details, fmt.Sprintf("RSI%.1f偏高有回调压力", rsi))
	}

	t0Dir := T0Skip
	switch {
	case t0Score >= 1:
		t0Dir = T0BuyFirst
	case t0Score <= -1:
		t0Dir = T0SellFirst
	}

	// === T+0 price targets (indicator-driven) ===
	t0Buy, t0Sell := calcT0Prices(
		price, ma5, ma20, rsi,
		bollUpper, bollLower, macdHist, prevMacdHist,
		quote,
	)

	// T+0 shares: 10% of holdings
	t0Shares := holdings / 10
	if t0Shares < 100 {
		t0Shares = 0
		t0Dir = T0Skip
		t0Details = []string{"底仓不足,无法做T+0"}
	}

	return SignalResult{
		Trend: trend, DCASignal: dcaSig,
		T0Dir: t0Dir, T0Shares: t0Shares,
		T0BuyPrice: t0Buy, T0SellPrice: t0Sell,
		T0Reasons: t0Details,
		Reason:    reasons,
	}
}
