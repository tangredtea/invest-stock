package backtest

import (
	"fmt"
	"time"

	"invest/pkg/data"
	"invest/pkg/indicator"
)

// platformMinBars is the platform hard lower bound on participating bars,
// matching the existing RunBacktest threshold (Requirement 19.8).
const platformMinBars = 60

// Engine is a stateless, deterministic backtest executor (Requirements 5.7, 5.8).
type Engine struct{}

// Run executes one backtest (Requirements 5.1, 5.2, 5.6, 5.9, 5.10). Any
// validation failure returns an error and produces no Result.
func (e Engine) Run(klines []data.KLine, s Strategy, cfg Config) (Result, error) {
	// 1. Strategy and config validation (Requirement 5.9).
	if s == nil {
		return Result{}, fmt.Errorf("策略不能为空")
	}
	if err := cfg.Validate(); err != nil {
		return Result{}, err
	}

	// 2. K线 time-series validation: finite OHLCV and strictly ascending dates
	// (Requirement 5.10).
	if err := data.ValidateKLines("K线", klines); err != nil {
		return Result{}, err
	}

	// 3. Date-range filtering (Requirements 6.3–6.6).
	window := filterByDate(klines, cfg.StartDate, cfg.EndDate)
	if (cfg.StartDate != nil || cfg.EndDate != nil) && len(window) == 0 {
		return Result{}, fmt.Errorf("指定日期区间内无可用 K线数据")
	}

	// 4. Minimum-bar validation (Requirements 5.5, 19.8).
	need := s.MinBars()
	if need < platformMinBars {
		need = platformMinBars
	}
	if len(window) < need {
		return Result{}, fmt.Errorf("k线数量不足: 实际 %d 根, 至少需要 %d 根", len(window), need)
	}

	// 5. Indicator precompute, aligned with the window.
	closes := make([]float64, len(window))
	for i, k := range window {
		closes[i] = k.Close
	}
	series := indicator.ComputeSeries(closes)

	// 6. Event loop.
	return e.runLoop(window, series, s, cfg)
}

// filterByDate returns the sub-slice whose Date falls in the optional [start,end]
// closed interval (Requirements 6.3, 6.4).
func filterByDate(klines []data.KLine, start, end *time.Time) []data.KLine {
	if start == nil && end == nil {
		return klines
	}
	out := make([]data.KLine, 0, len(klines))
	for _, k := range klines {
		if start != nil && k.Date.Before(*start) {
			continue
		}
		if end != nil && k.Date.After(*end) {
			continue
		}
		out = append(out, k)
	}
	return out
}

// runLoop drives the strategy bar by bar, matching fills per the Fill rule,
// applying costs and T+1, recording trades and curves, then computing metrics.
func (e Engine) runLoop(window []data.KLine, series indicator.Series, s Strategy, cfg Config) (Result, error) {
	n := len(window)
	acc := newAccount(cfg.InitialCash)
	curves := newEngineCurves(n)
	var unfilled []UnfilledOrder

	peak := cfg.InitialCash

	for i := 0; i < n; i++ {
		ctx := &Context{
			Index: i, Klines: window, Series: series,
			Cash: acc.cash, Shares: acc.shares(),
		}
		dec := s.Decide(ctx)
		if err := validateDecision(dec); err != nil {
			return Result{}, fmt.Errorf("第 %d 根 K线策略决策非法: %w", i, err)
		}

		if dec.Action != Hold {
			e.execute(acc, &unfilled, window, i, dec, cfg)
		}

		peak = curves.record(i, window[i], acc, peak, cfg.InitialCash)
	}

	return buildEngineResult(s, cfg, window, acc, curves, unfilled), nil
}
