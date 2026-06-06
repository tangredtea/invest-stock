package builtin

import "invest/pkg/backtest"

// multiFactor: MA/MACD/RSI/Boll scoring; score >= 2 -> full buy; score <= -2 -> full sell.
type multiFactor struct {
	startIndex int
}

func (s *multiFactor) Name() string { return NameMultiFactor }
func (s *multiFactor) MinBars() int { return s.startIndex + 1 }
func (s *multiFactor) Params() []backtest.ParamSpec {
	return []backtest.ParamSpec{startIndexSpec(), initialCashSpec()}
}
func (s *multiFactor) Decide(ctx *backtest.Context) backtest.Decision {
	i := ctx.Index
	if i < s.startIndex+1 {
		return backtest.HoldDecision()
	}
	price := ctx.Bar(i).Close
	score := 0
	if ctx.Ind(backtest.FieldMA5, i) > ctx.Ind(backtest.FieldMA20, i) {
		score++
	} else {
		score--
	}
	if ctx.Ind(backtest.FieldMACDHist, i) > 0 {
		score++
	} else {
		score--
	}
	rsi := ctx.Ind(backtest.FieldRSI14, i)
	if rsi > 0 && rsi < 40 {
		score++
	} else if rsi > 60 {
		score--
	}
	lower := ctx.Ind(backtest.FieldBollLower, i)
	upper := ctx.Ind(backtest.FieldBollUpper, i)
	if lower > 0 && price <= lower {
		score += 2
	} else if upper > 0 && price >= upper {
		score -= 2
	}

	if score >= 2 && ctx.Shares == 0 {
		return backtest.BuyAmount(ctx.Cash)
	}
	if score <= -2 && ctx.Shares > 0 {
		return backtest.SellQty(ctx.Shares)
	}
	return backtest.HoldDecision()
}

func newMultiFactor(p map[string]backtest.ParamValue) (backtest.Strategy, error) {
	return &multiFactor{startIndex: paramInt(p, "startIndex", defaultStartIndex)}, nil
}

// grid: trade around the Bollinger middle band in 0.5%-spaced grid levels.
// Mirrors the legacy btGrid: posPerLevel = initialCash/gridLevels; price below
// the current level -> add a level, above -> reduce a level.
type grid struct {
	startIndex  int
	initialCash float64
	gridLevels  int
	currentLevel int
}

func (s *grid) Name() string { return NameGrid }
func (s *grid) MinBars() int { return s.startIndex + 1 }
func (s *grid) Params() []backtest.ParamSpec {
	return []backtest.ParamSpec{
		startIndexSpec(), initialCashSpec(),
		{Name: "gridLevels", Type: backtest.ParamInt, Default: backtest.IntVal(5), Min: 1, Max: 50, Desc: "网格层数"},
	}
}
func (s *grid) Decide(ctx *backtest.Context) backtest.Decision {
	i := ctx.Index
	if i < s.startIndex {
		return backtest.HoldDecision()
	}
	mid := ctx.Ind(backtest.FieldBollMid, i)
	if mid == 0 {
		return backtest.HoldDecision()
	}
	price := ctx.Bar(i).Close
	dev := (price - mid) / mid
	target := int(dev / 0.005)
	if target > s.gridLevels {
		target = s.gridLevels
	} else if target < -s.gridLevels {
		target = -s.gridLevels
	}
	posPerLevel := s.initialCash / float64(s.gridLevels)

	// Price fell relative to level -> buy one level's worth.
	if s.currentLevel > target && ctx.Cash >= posPerLevel {
		s.currentLevel--
		return backtest.BuyAmount(posPerLevel)
	}
	// Price rose relative to level -> sell one level's worth.
	if s.currentLevel < target && ctx.Shares > 0 {
		s.currentLevel++
		qty := int(posPerLevel / price)
		if qty > ctx.Shares {
			qty = ctx.Shares
		}
		if qty > 0 {
			return backtest.SellQty(qty)
		}
	}
	return backtest.HoldDecision()
}

func newGrid(p map[string]backtest.ParamValue) (backtest.Strategy, error) {
	return &grid{
		startIndex:  paramInt(p, "startIndex", defaultStartIndex),
		initialCash: paramFloat(p, "initialCash", defaultInitialCash),
		gridLevels:  paramInt(p, "gridLevels", 5),
	}, nil
}
