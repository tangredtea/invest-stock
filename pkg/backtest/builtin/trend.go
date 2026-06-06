package builtin

import "invest/pkg/backtest"

// buyHold buys the full position at startIndex and holds to the end.
type buyHold struct {
	startIndex int
	bought     bool
}

func (s *buyHold) Name() string  { return NameBuyHold }
func (s *buyHold) MinBars() int  { return s.startIndex + 1 }
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

// maCross: MA5 crossing above MA20 -> full buy; crossing below -> full sell.
type maCross struct {
	startIndex int
}

func (s *maCross) Name() string  { return NameMACross }
func (s *maCross) MinBars() int  { return s.startIndex + 1 }
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
