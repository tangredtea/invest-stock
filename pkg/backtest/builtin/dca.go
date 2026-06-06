package builtin

import (
	"time"

	"invest/pkg/backtest"
)

// weekdayFromString maps an English weekday abbreviation to time.Weekday.
func weekdayFromString(s string) time.Weekday {
	switch s {
	case "Mon":
		return time.Monday
	case "Tue":
		return time.Tuesday
	case "Wed":
		return time.Wednesday
	case "Thu":
		return time.Thursday
	case "Fri":
		return time.Friday
	case "Sat":
		return time.Saturday
	case "Sun":
		return time.Sunday
	default:
		return time.Wednesday
	}
}

// dcaT0: dollar-cost averaging — buy a fixed weekly budget on the DCA weekday.
//
// Note: the legacy "智能定投+T0" added a synthetic intraday T+0 profit directly
// to cash. That is a non-physical cash injection (not a real fill), so the new
// engine models only the realistic DCA buys. The synthetic T+0 cash adjustment
// is intentionally not reproduced (see compat layer notes).
type dcaT0 struct {
	startIndex   int
	weeklyBudget float64
	weekday      time.Weekday
}

func (s *dcaT0) Name() string { return NameDCAEnhanced }
func (s *dcaT0) MinBars() int { return s.startIndex + 1 }
func (s *dcaT0) Params() []backtest.ParamSpec {
	return []backtest.ParamSpec{startIndexSpec(), weeklyBudgetSpec(), dcaWeekdaySpec()}
}
func (s *dcaT0) Decide(ctx *backtest.Context) backtest.Decision {
	i := ctx.Index
	if i < s.startIndex {
		return backtest.HoldDecision()
	}
	if ctx.Bar(i).Date.Weekday() == s.weekday && ctx.Cash >= s.weeklyBudget {
		return backtest.BuyAmount(s.weeklyBudget)
	}
	return backtest.HoldDecision()
}

func newDCAT0(p map[string]backtest.ParamValue) (backtest.Strategy, error) {
	return &dcaT0{
		startIndex:   paramInt(p, "startIndex", defaultStartIndex),
		weeklyBudget: paramFloat(p, "weeklyBudget", defaultWeeklyBudget),
		weekday:      weekdayFromString(paramStr(p, "dcaWeekday", defaultDCAWeekday)),
	}, nil
}
