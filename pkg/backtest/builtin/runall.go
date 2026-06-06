package builtin

import (
	"invest/pkg/backtest"
	"invest/pkg/data"
)

// Summary is a compact per-strategy result for the analyze view and CLI.
// It mirrors the legacy BacktestResult field set so existing consumers keep
// working, but the values come from the new, rigorous engine.
type Summary struct {
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

// RunAll runs all 9 builtin strategies on the given klines using the default
// config and returns a compact summary per strategy, in canonical order.
// Returns an empty slice when there are too few bars for the engine.
func RunAll(klines []data.KLine) []Summary {
	eng := backtest.Engine{}
	cfg := backtest.DefaultConfig()
	out := make([]Summary, 0, 9)
	for _, name := range Names() {
		s, err := backtest.Default.New(name, nil)
		if err != nil {
			continue
		}
		res, err := eng.Run(klines, s, cfg)
		if err != nil {
			// Too few bars (or other validation): match legacy "return nil".
			return nil
		}
		out = append(out, summarize(res, cfg.InitialCash))
	}
	return out
}

// summarize projects a full engine Result onto the compact Summary shape.
func summarize(res backtest.Result, initialCash float64) Summary {
	finalShares := 0
	totalBuyTurnover := 0.0
	for _, t := range res.Trades {
		if t.Action == "buy" {
			finalShares += t.Qty
			totalBuyTurnover += t.Turnover
		} else if t.Action == "sell" {
			finalShares -= t.Qty
		}
	}
	if finalShares < 0 {
		finalShares = 0
	}
	finalValue := 0.0
	if len(res.Equity) > 0 {
		finalValue = res.Equity[len(res.Equity)-1]
	}
	avgCost := 0.0
	if finalShares > 0 {
		avgCost = totalBuyTurnover / float64(finalShares)
	}
	return Summary{
		Name:         res.StrategyName,
		TotalReturn:  res.Metrics.TotalReturn,
		AnnualReturn: res.Metrics.AnnualReturn,
		MaxDrawdown:  res.Metrics.MaxDrawdown,
		SharpeRatio:  res.Metrics.Sharpe,
		TotalTrades:  res.Metrics.TotalTrades,
		FinalShares:  finalShares,
		FinalValue:   finalValue,
		TotalCost:    initialCash,
		AvgCost:      avgCost,
	}
}
