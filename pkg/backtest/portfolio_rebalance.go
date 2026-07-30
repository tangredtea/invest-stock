package backtest

import "math"

// shouldRebalance reports whether to rebalance at bar t (periodic or threshold),
// collapsing simultaneous triggers into one (Requirements 4.1, 4.2, 4.5).
func (e PortfolioEngine) shouldRebalance(cfg PortfolioConfig, acc *portfolioAccount, prices, targetW map[string]float64, t, lastIdx int) (do, isReb bool) {
	if cfg.Rebalance.Periodic && t > 0 && (t-lastIdx) >= cfg.Rebalance.PeriodBars {
		return true, true
	}
	if cfg.Rebalance.Threshold {
		for sym, tw := range targetW {
			cur := acc.weight(sym, prices)
			if math.Abs(cur-tw) > cfg.Rebalance.ThresholdValue {
				return true, true
			}
		}
	}
	return false, false
}
