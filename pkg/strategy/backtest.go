package strategy

import (
	"fmt"

	"invest/pkg/backtest/builtin"
	"invest/pkg/data"
)

// BacktestResult is the compact per-strategy summary returned by RunBacktest
// and embedded in the analyze response. Field names/types are unchanged for the
// frontend and CLI; values come from the rigorous new backtest engine.
type BacktestResult struct {
	Name         string  `json:"name"`
	TotalReturn  float64 `json:"totalReturn"`
	AnnualReturn float64 `json:"annualReturn"`
	MaxDrawdown  float64 `json:"maxDrawdown"`
	SharpeRatio  float64 `json:"sharpeRatio"`
	TotalTrades  int     `json:"totalTrades"`
	FinalShares  int     `json:"finalShares"`
	FinalValue   float64 `json:"finalValue"`
	TotalCost    float64 `json:"totalCost"`
	AvgCost      float64 `json:"avgCost"`
}

func (r BacktestResult) String() string {
	return fmt.Sprintf("%-24s | 收益: %6.2f%% | 年化: %6.2f%% | 回撤: %5.2f%% | 夏普: %5.2f | 持仓: %6d | 均价: %.4f | 终值: %9.0f | 投入: %9.0f",
		r.Name, r.TotalReturn*100, r.AnnualReturn*100, r.MaxDrawdown*100,
		r.SharpeRatio, r.FinalShares, r.AvgCost, r.FinalValue, r.TotalCost)
}

// RunBacktest runs the 9 builtin strategies via the new engine and returns a
// compact summary per strategy. Returns nil when there are too few bars.
func RunBacktest(klines []data.KLine) []BacktestResult {
	summaries := builtin.RunAll(klines)
	if len(summaries) == 0 {
		return nil
	}
	out := make([]BacktestResult, len(summaries))
	for i, s := range summaries {
		out[i] = BacktestResult(s)
	}
	return out
}
