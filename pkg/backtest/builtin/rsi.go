package builtin

import "invest/pkg/backtest"

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
