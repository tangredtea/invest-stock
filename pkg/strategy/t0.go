package strategy

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
		// MA5 below price -> support; above price -> resistance
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

	// 4. RSI adjustment: low RSI raises buy price (more aggressive buy);
	// high RSI lowers sell price (more aggressive sell).
	if rsi > 0 {
		rsiAdj := (50 - rsi) / 500 // +/-0.1 range for RSI 0-100
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
