package builtin

import "invest/pkg/backtest"

// grid: trade around the Bollinger middle band in 0.5%-spaced grid levels.
// Mirrors the legacy btGrid: posPerLevel = initialCash/gridLevels; price below
// the current level -> add a level, above -> reduce a level.
type grid struct {
	startIndex   int
	initialCash  float64
	gridLevels   int
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
