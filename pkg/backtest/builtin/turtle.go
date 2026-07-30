package builtin

import "invest/pkg/backtest"

// turtle: price > prior 20-bar high -> full buy; price < prior 10-bar low -> full sell.
type turtle struct {
	startIndex            int
	highWindow, lowWindow int
}

func (s *turtle) Name() string { return NameTurtle }
func (s *turtle) MinBars() int { return s.startIndex + 1 }
func (s *turtle) Params() []backtest.ParamSpec {
	return []backtest.ParamSpec{
		startIndexSpec(), initialCashSpec(),
		{Name: "highWindow", Type: backtest.ParamInt, Default: backtest.IntVal(20), Min: 1, Max: 250, Desc: "突破最高价窗口"},
		{Name: "lowWindow", Type: backtest.ParamInt, Default: backtest.IntVal(10), Min: 1, Max: 250, Desc: "跌破最低价窗口"},
	}
}
func (s *turtle) Decide(ctx *backtest.Context) backtest.Decision {
	i := ctx.Index
	if i < s.startIndex {
		return backtest.HoldDecision()
	}
	price := ctx.Bar(i).Close
	high := 0.0
	for j := i - s.highWindow; j < i; j++ {
		if j >= 0 && ctx.Bar(j).High > high {
			high = ctx.Bar(j).High
		}
	}
	low := price
	for j := i - s.lowWindow; j < i; j++ {
		if j >= 0 && ctx.Bar(j).Low < low {
			low = ctx.Bar(j).Low
		}
	}
	if price > high && ctx.Shares == 0 {
		return backtest.BuyAmount(ctx.Cash)
	}
	if price < low && ctx.Shares > 0 {
		return backtest.SellQty(ctx.Shares)
	}
	return backtest.HoldDecision()
}

func newTurtle(p map[string]backtest.ParamValue) (backtest.Strategy, error) {
	return &turtle{
		startIndex: paramInt(p, "startIndex", defaultStartIndex),
		highWindow: paramInt(p, "highWindow", 20),
		lowWindow:  paramInt(p, "lowWindow", 10),
	}, nil
}
