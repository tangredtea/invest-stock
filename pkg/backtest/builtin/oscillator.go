package builtin

import "invest/pkg/backtest"

// bollinger: price <= lower band -> full buy; price >= upper band -> full sell.
type bollinger struct {
	startIndex int
}

func (s *bollinger) Name() string { return NameBollinger }
func (s *bollinger) MinBars() int { return s.startIndex + 1 }
func (s *bollinger) Params() []backtest.ParamSpec {
	return []backtest.ParamSpec{startIndexSpec(), initialCashSpec()}
}
func (s *bollinger) Decide(ctx *backtest.Context) backtest.Decision {
	i := ctx.Index
	if i < s.startIndex {
		return backtest.HoldDecision()
	}
	lower := ctx.Ind(backtest.FieldBollLower, i)
	upper := ctx.Ind(backtest.FieldBollUpper, i)
	if lower == 0 {
		return backtest.HoldDecision()
	}
	price := ctx.Bar(i).Close
	if price <= lower && ctx.Shares == 0 {
		return backtest.BuyAmount(ctx.Cash)
	}
	if price >= upper && ctx.Shares > 0 {
		return backtest.SellQty(ctx.Shares)
	}
	return backtest.HoldDecision()
}

func newBollinger(p map[string]backtest.ParamValue) (backtest.Strategy, error) {
	return &bollinger{startIndex: paramInt(p, "startIndex", defaultStartIndex)}, nil
}

// rsiStrat: RSI < buyLevel -> full buy; RSI > sellLevel -> full sell.
type rsiStrat struct {
	startIndex          int
	buyLevel, sellLevel float64
}

func (s *rsiStrat) Name() string { return NameRSI }
func (s *rsiStrat) MinBars() int { return s.startIndex + 1 }
func (s *rsiStrat) Params() []backtest.ParamSpec {
	return []backtest.ParamSpec{
		startIndexSpec(), initialCashSpec(),
		{Name: "buyLevel", Type: backtest.ParamFloat, Default: backtest.FloatVal(30), Min: 0, Max: 100, Desc: "RSI 买入阈值"},
		{Name: "sellLevel", Type: backtest.ParamFloat, Default: backtest.FloatVal(70), Min: 0, Max: 100, Desc: "RSI 卖出阈值"},
	}
}
func (s *rsiStrat) Decide(ctx *backtest.Context) backtest.Decision {
	i := ctx.Index
	if i < s.startIndex {
		return backtest.HoldDecision()
	}
	rsi := ctx.Ind(backtest.FieldRSI14, i)
	if rsi == 0 {
		return backtest.HoldDecision()
	}
	if rsi < s.buyLevel && ctx.Shares == 0 {
		return backtest.BuyAmount(ctx.Cash)
	}
	if rsi > s.sellLevel && ctx.Shares > 0 {
		return backtest.SellQty(ctx.Shares)
	}
	return backtest.HoldDecision()
}

func newRSI(p map[string]backtest.ParamValue) (backtest.Strategy, error) {
	return &rsiStrat{
		startIndex: paramInt(p, "startIndex", defaultStartIndex),
		buyLevel:   paramFloat(p, "buyLevel", 30),
		sellLevel:  paramFloat(p, "sellLevel", 70),
	}, nil
}

// turtle: price > prior 20-bar high -> full buy; price < prior 10-bar low -> full sell.
type turtle struct {
	startIndex          int
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
