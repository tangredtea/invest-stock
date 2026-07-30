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
