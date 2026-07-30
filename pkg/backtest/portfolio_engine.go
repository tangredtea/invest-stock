package backtest

import (
	"fmt"
)

// PortfolioEngine is a stateless, deterministic portfolio backtest executor
// (Requirements 3.9, 3.10).
type PortfolioEngine struct{}

// Run executes one portfolio backtest (Requirement 3). Any validation failure
// returns an error and produces no PortfolioResult.
func (e PortfolioEngine) Run(aligned AlignedData, s PortfolioStrategy, cfg PortfolioConfig) (PortfolioResult, error) {
	if s == nil {
		return PortfolioResult{}, fmt.Errorf("组合策略不能为空")
	}
	if err := cfg.Validate(); err != nil {
		return PortfolioResult{}, err
	}
	if err := validateAlignedData(aligned); err != nil {
		return PortfolioResult{}, err
	}

	symbols := aligned.Symbols
	allowedSymbols := symbolSet(symbols)
	if err := validatePortfolioConfigSymbols(cfg, allowedSymbols); err != nil {
		return PortfolioResult{}, err
	}

	n := len(aligned.Timeline)
	if n < platformMinBars {
		return PortfolioResult{}, fmt.Errorf("时间轴长度不足: 实际 %d, 至少需要 %d", n, platformMinBars)
	}

	series := computePortfolioIndicatorSeries(aligned, symbols)

	acc := newPortfolioAccount(cfg.Base.InitialCash, symbols)
	curves := newPortfolioCurves(symbols, n)
	peakClose := make(map[string]float64) // highest close since entry, for stop-loss
	var unfilled []PortfolioUnfilled

	peak := cfg.Base.InitialCash
	lastRebalanceIdx := 0
	rebalanceCount := 0
	var rebalanceCost float64
	var pending []order // orders decided at t-1, executed at t's open (FillNextOpen)

	for t := 0; t < n; t++ {
		validCloses := e.validCloses(aligned, t)

		// Execute pending orders (decided last bar) at THIS bar's open under
		// the next-open fill rule, before any new decision. This keeps
		// equity[0] == initial cash (Requirement 3.8).
		if cfg.Base.FillRule == FillNextOpen && len(pending) > 0 {
			rebalanceCost += e.matchOrders(pending, acc, aligned, validCloses, cfg, t, &unfilled, priceOpen)
			pending = nil
		}

		ctx := newPortfolioContext(t, aligned, series, acc, validCloses)

		dec := s.Decide(ctx)
		if err := validatePortfolioDecision(dec); err != nil {
			return PortfolioResult{}, fmt.Errorf("第 %d 个时间点组合策略决策非法: %w", t, err)
		}
		if dec.Kind == DecisionSignals {
			if err := rejectUnknownSignals(dec.Signals, allowedSymbols); err != nil {
				return PortfolioResult{}, fmt.Errorf("第 %d 个时间点组合策略决策非法: %w", t, err)
			}
		}

		// Resolve this bar's target weights (Requirement 5.6, 5.7).
		targetW, err := e.resolveTargetWeights(dec, ctx, aligned, series, cfg, t, allowedSymbols)
		if err != nil {
			return PortfolioResult{}, err
		}

		// Decide whether to act this bar.
		doRebalance, isRebalance := e.shouldRebalance(cfg, acc, validCloses, targetW, t, lastRebalanceIdx)

		var orders []order
		if dec.Kind == DecisionSignals {
			orders = e.signalOrders(dec, acc, aligned, validCloses, cfg, t)
		} else if t == 0 || doRebalance {
			orders = e.weightOrders(targetW, acc, validCloses, cfg, t)
			if doRebalance {
				isRebalance = true
			}
		}

		// Risk hooks: stop-loss / take-profit / max-drawdown forced orders.
		e.updatePeakClose(peakClose, acc, validCloses)
		riskOrders := e.riskOrders(cfg, acc, aligned, validCloses, peakClose, peak, t)
		orders = append(orders, riskOrders...)

		if isRebalance && len(orders) > 0 {
			rebalanceCount++
			lastRebalanceIdx = t
		}

		// Execute per fill rule: close => now at this bar's close;
		// next-open => queue as pending for next bar's open (Requirement 5.3).
		if cfg.Base.FillRule == FillClose {
			rebalanceCost += e.matchOrders(orders, acc, aligned, validCloses, cfg, t, &unfilled, priceClose)
		} else {
			pending = orders
		}

		peak = curves.record(t, acc, symbols, validCloses, peak, cfg.Base.InitialCash)
	}

	// Any pending orders at the end (next-open with no next bar) are dropped as
	// unfilled (Requirement 5.4).
	for _, o := range pending {
		unfilled = append(unfilled, PortfolioUnfilled{Symbol: o.sym, UnfilledOrder: UnfilledOrder{Date: aligned.Timeline[n-1], Action: o.action.String(), Reason: "no_next_bar", Qty: o.qty}})
	}

	res := e.buildPortfolioResult(aligned, symbols, cfg, acc, curves, unfilled, rebalanceCount, rebalanceCost)
	if cfg.Correlation {
		res.Correlation = correlationMatrix(aligned, symbols)
	}
	return res, nil
}

// order is an internal buy/sell instruction for one symbol.
type order struct {
	sym       string
	action    Action
	qty       int
	rebalance bool
}

// validCloses returns each symbol's effective close at index t (last valid close
// for suspended symbols; 0 for missing) (Requirement 1.5).
func (e PortfolioEngine) validCloses(aligned AlignedData, t int) map[string]float64 {
	out := make(map[string]float64, len(aligned.Symbols))
	for _, sym := range aligned.Symbols {
		out[sym] = aligned.Points[sym][t].Close
	}
	return out
}
