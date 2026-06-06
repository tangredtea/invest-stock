// Package builtin migrates the 9 existing backtest strategies onto the unified
// Strategy interface and registers them with backtest.Default.
//
// Each strategy reads only data at indices <= the current bar (no look-ahead),
// and its default parameters reproduce the original hardcoded constants. The new
// engine is the single source of truth for backtesting; RunBacktest delegates to
// these strategies via RunAll.
package builtin

import "invest/pkg/backtest"

// Strategy names — kept identical to the legacy display names so the analyze
// view and CLI render the same labels (Requirement 3.1).
const (
	NameBuyHold     = "买入持有(基准)"
	NameMACross     = "双均线交叉(MA5/20)"
	NameMACDCross   = "MACD金叉死叉"
	NameBollinger   = "布林带均值回归"
	NameRSI         = "RSI超买超卖(30/70)"
	NameTurtle      = "海龟突破(20/10)"
	NameGrid        = "网格交易(布林中轨)"
	NameMultiFactor = "多因子综合"
	NameDCAEnhanced = "智能定投+T0"
)

// Default constants reproduced from the legacy implementation (Requirement 3.3).
const (
	defaultStartIndex  = 60
	defaultInitialCash = 100000.0
	defaultWeeklyBudget = 2000.0
	defaultDCAWeekday   = "Wed"
)

// startIndexSpec returns the shared "startIndex" parameter spec.
func startIndexSpec() backtest.ParamSpec {
	return backtest.ParamSpec{
		Name: "startIndex", Type: backtest.ParamInt,
		Default: backtest.IntVal(defaultStartIndex), Min: 0, Max: 100000,
		Desc: "回测起始 K线索引",
	}
}

// initialCashSpec returns the shared "initialCash" parameter spec.
func initialCashSpec() backtest.ParamSpec {
	return backtest.ParamSpec{
		Name: "initialCash", Type: backtest.ParamFloat,
		Default: backtest.FloatVal(defaultInitialCash), Min: 0.01, Max: 999999999.99,
		Desc: "初始资金(元)",
	}
}

// weeklyBudgetSpec returns the shared "weeklyBudget" parameter spec.
func weeklyBudgetSpec() backtest.ParamSpec {
	return backtest.ParamSpec{
		Name: "weeklyBudget", Type: backtest.ParamFloat,
		Default: backtest.FloatVal(defaultWeeklyBudget), Min: 0.01, Max: 999999999.99,
		Desc: "每周定投预算(元)",
	}
}

// dcaWeekdaySpec returns the shared "dcaWeekday" enum parameter spec.
func dcaWeekdaySpec() backtest.ParamSpec {
	return backtest.ParamSpec{
		Name: "dcaWeekday", Type: backtest.ParamEnum,
		Default: backtest.EnumVal(defaultDCAWeekday),
		Enum:    []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
		Desc:    "定投触发日(星期)",
	}
}

// paramInt / paramFloat / paramStr read a validated param with a fallback.
func paramInt(p map[string]backtest.ParamValue, name string, def int) int {
	if v, ok := p[name]; ok {
		return v.Int
	}
	return def
}

func paramFloat(p map[string]backtest.ParamValue, name string, def float64) float64 {
	if v, ok := p[name]; ok {
		return v.Float
	}
	return def
}

func paramStr(p map[string]backtest.ParamValue, name, def string) string {
	if v, ok := p[name]; ok {
		return v.Str
	}
	return def
}
