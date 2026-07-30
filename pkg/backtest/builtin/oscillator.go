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
