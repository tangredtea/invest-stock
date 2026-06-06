package strategy

import "fmt"

type Signal int

const (
	StrongBuy  Signal = 2
	Buy        Signal = 1
	Hold       Signal = 0
	Sell       Signal = -1
	StrongSell Signal = -2
)

func (s Signal) String() string {
	switch s {
	case StrongBuy:
		return "强买入"
	case Buy:
		return "买入"
	case Hold:
		return "持有"
	case Sell:
		return "卖出"
	case StrongSell:
		return "强卖出"
	default:
		return "未知"
	}
}

// T0Direction indicates intraday T+0 operation direction.
type T0Direction int

const (
	T0BuyFirst  T0Direction = 1  // 先买后卖(看涨日内)
	T0SellFirst T0Direction = -1 // 先卖后买(看跌日内)
	T0Skip      T0Direction = 0  // 不做T+0
)

func (d T0Direction) String() string {
	switch d {
	case T0BuyFirst:
		return "正T(先买后卖)"
	case T0SellFirst:
		return "反T(先卖后买)"
	default:
		return "观望不做"
	}
}

type SignalResult struct {
	Trend       Trend       `json:"trend"`
	DCASignal   Signal      `json:"dcaSignal"`
	T0Dir       T0Direction `json:"t0Dir"`
	T0Shares    int         `json:"t0Shares"`
	T0BuyPrice  float64     `json:"t0BuyPrice"`
	T0SellPrice float64     `json:"t0SellPrice"`
	T0Reasons   []string    `json:"t0Reasons"`
	Reason      string      `json:"reason"`
}

// QuoteInfo holds real-time intraday data for T+0 price calculation.
type QuoteInfo struct {
	Price, High, Low, PreClose float64
}

// calcT0Prices computes indicator-driven T+0 buy/sell prices.
// Each indicator contributes a support/resistance level with a weight.
// Final prices are weighted averages, clamped to intraday range.
func calcT0Prices(price, ma5, ma20, rsi, bollUpper, bollLower, macdHist, prevMacdHist float64,
	q QuoteInfo) (buyAt, sellAt float64) {

	type level struct {
		support, resist, weight float64
	}
	var levels []level

	// 1. MA5 as short-term support/resistance
	if ma5 > 0 {
		// MA5 below price → support; above price → resistance
		levels = append(levels, level{
			support: ma5 - (price-ma5)*0.3,
			resist:  ma5 + (price-ma5)*0.3,
			weight:  2.0, // MA5 is most relevant for intraday
		})
	}

	// 2. Bollinger bands
	if bollLower > 0 && bollUpper > 0 {
		mid := (bollUpper + bollLower) / 2
		levels = append(levels, level{
			support: bollLower + (mid-bollLower)*0.3,
			resist:  mid + (bollUpper-mid)*0.3,
			weight:  1.5,
		})
	}

	// 3. MA20 as medium-term anchor
	if ma20 > 0 {
		dev := price - ma20
		levels = append(levels, level{
			support: ma20 + dev*0.3,
			resist:  ma20 + dev*1.5,
			weight:  1.0,
		})
	}

	// Weighted average
	var sumBuy, sumSell, sumW float64
	for _, l := range levels {
		sumBuy += l.support * l.weight
		sumSell += l.resist * l.weight
		sumW += l.weight
	}
	if sumW == 0 {
		return 0, 0
	}
	buyAt = sumBuy / sumW
	sellAt = sumSell / sumW

	// 4. RSI adjustment: low RSI → raise buy price (more aggressive buy);
	//    high RSI → lower sell price (more aggressive sell)
	if rsi > 0 {
		rsiAdj := (50 - rsi) / 500 // ±0.1 range for RSI 0-100
		buyAt *= (1 + rsiAdj)
		sellAt *= (1 - rsiAdj)
	}

	// 5. MACD momentum shift
	if prevMacdHist != 0 {
		macdDelta := macdHist - prevMacdHist
		momentum := macdDelta / price * 10 // normalize
		if momentum > 0.02 {
			momentum = 0.02
		} else if momentum < -0.02 {
			momentum = -0.02
		}
		buyAt *= (1 + momentum)
		sellAt *= (1 + momentum)
	}

	// 6. Clamp to intraday range if we have real-time data
	if q.High > 0 && q.Low > 0 {
		if buyAt < q.Low {
			buyAt = q.Low
		}
		if sellAt > q.High*1.005 { // allow slight overshoot
			sellAt = q.High * 1.005
		}
		if buyAt >= sellAt {
			// ensure spread exists: split around current price
			spread := q.High - q.Low
			if spread < 0.001 {
				spread = price * 0.01
			}
			buyAt = price - spread*0.3
			sellAt = price + spread*0.3
		}
	}

	return buyAt, sellAt
}

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
