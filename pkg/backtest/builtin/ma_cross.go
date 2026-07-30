package builtin

import "invest/pkg/backtest"

// maCross: MA5 crossing above MA20 -> full buy; crossing below -> full sell.
type maCross struct {
	startIndex int
}

func (s *maCross) Name() string { return NameMACross }
func (s *maCross) MinBars() int { return s.startIndex + 1 }
func (s *maCross) Params() []backtest.ParamSpec {
	return []backtest.ParamSpec{startIndexSpec(), initialCashSpec()}
}
func (s *maCross) Decide(ctx *backtest.Context) backtest.Decision {
	i := ctx.Index
	if i < s.startIndex+1 {
		return backtest.HoldDecision()
	}
	prevAbove := ctx.Ind(backtest.FieldMA5, i-1) > ctx.Ind(backtest.FieldMA20, i-1)
	currAbove := ctx.Ind(backtest.FieldMA5, i) > ctx.Ind(backtest.FieldMA20, i)
	if !prevAbove && currAbove && ctx.Shares == 0 {
		return backtest.BuyAmount(ctx.Cash)
	}
	if prevAbove && !currAbove && ctx.Shares > 0 {
		return backtest.SellQty(ctx.Shares)
	}
	return backtest.HoldDecision()
}

func newMACross(p map[string]backtest.ParamValue) (backtest.Strategy, error) {
	return &maCross{startIndex: paramInt(p, "startIndex", defaultStartIndex)}, nil
}
