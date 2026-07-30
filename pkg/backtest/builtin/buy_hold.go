package builtin

import "invest/pkg/backtest"

// buyHold buys the full position at startIndex and holds to the end.
type buyHold struct {
	startIndex int
	bought     bool
}

func (s *buyHold) Name() string { return NameBuyHold }
func (s *buyHold) MinBars() int { return s.startIndex + 1 }
func (s *buyHold) Params() []backtest.ParamSpec {
	return []backtest.ParamSpec{startIndexSpec(), initialCashSpec()}
}
func (s *buyHold) Decide(ctx *backtest.Context) backtest.Decision {
	if ctx.Index == s.startIndex && !s.bought && ctx.Shares == 0 {
		s.bought = true
		return backtest.BuyAmount(ctx.Cash)
	}
	return backtest.HoldDecision()
}

func newBuyHold(p map[string]backtest.ParamValue) (backtest.Strategy, error) {
	return &buyHold{startIndex: paramInt(p, "startIndex", defaultStartIndex)}, nil
}
