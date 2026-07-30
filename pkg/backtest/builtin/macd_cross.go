package builtin

import "invest/pkg/backtest"

// macdCross: MACD histogram crossing up through 0 -> buy; down through 0 -> sell.
type macdCross struct {
	startIndex int
}

func (s *macdCross) Name() string { return NameMACDCross }
func (s *macdCross) MinBars() int { return s.startIndex + 1 }
func (s *macdCross) Params() []backtest.ParamSpec {
	return []backtest.ParamSpec{startIndexSpec(), initialCashSpec()}
}
func (s *macdCross) Decide(ctx *backtest.Context) backtest.Decision {
	i := ctx.Index
	if i < s.startIndex+1 {
		return backtest.HoldDecision()
	}
	prev := ctx.Ind(backtest.FieldMACDHist, i-1)
	curr := ctx.Ind(backtest.FieldMACDHist, i)
	if prev <= 0 && curr > 0 && ctx.Shares == 0 {
		return backtest.BuyAmount(ctx.Cash)
	}
	if prev >= 0 && curr < 0 && ctx.Shares > 0 {
		return backtest.SellQty(ctx.Shares)
	}
	return backtest.HoldDecision()
}

func newMACDCross(p map[string]backtest.ParamValue) (backtest.Strategy, error) {
	return &macdCross{startIndex: paramInt(p, "startIndex", defaultStartIndex)}, nil
}
