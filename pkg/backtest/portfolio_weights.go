package backtest

import (
	"fmt"
	"math"

	"invest/pkg/indicator"
)

// resolveTargetWeights computes the target weights for bar t from the decision
// and the configured weight scheme, then applies position caps and cash reserve
// (Requirements 5.4, 5.5, 5.6, 5.7).
func (e PortfolioEngine) resolveTargetWeights(dec PortfolioDecision, ctx *PortfolioContext, aligned AlignedData, series map[string]indicator.Series, cfg PortfolioConfig, t int, allowed map[string]struct{}) (map[string]float64, error) {
	tradable := make([]string, 0, len(aligned.Symbols))
	for _, sym := range aligned.Symbols {
		if aligned.Points[sym][t].Valid {
			tradable = append(tradable, sym)
		}
	}

	w := make(map[string]float64)
	if dec.Kind == DecisionWeights && len(dec.Weights) > 0 {
		if err := rejectUnknownWeights(dec.Weights, allowed); err != nil {
			return nil, err
		}
		// Validate each weight in [0,1] and sum <= 1 (Requirements 2.9, 2.5).
		var sum float64
		for sym, weight := range dec.Weights {
			if math.IsNaN(weight) || math.IsInf(weight, 0) || weight < 0 || weight > 1 {
				return nil, fmt.Errorf("标的 %s 目标权重 %g 不在 [0,1]", sym, weight)
			}
			sum += weight
		}
		if sum > 1+weightSumTolerance {
			return nil, fmt.Errorf("目标权重之和 %g 超过 1", sum)
		}
		for sym, weight := range dec.Weights {
			w[sym] = weight
		}
	} else {
		// Signal mode, or a target-weight strategy deferring to the configured
		// scheme (nil weights): weights come from the scheme (Requirement 5.6).
		w = e.schemeWeights(cfg, tradable, aligned, series, t)
	}

	// Apply per-symbol cap (Requirement 5.4).
	for sym := range w {
		if cap, ok := cfg.Risk.PerSymbolCap[sym]; ok && w[sym] > cap {
			w[sym] = cap
		}
	}
	// Apply cash reserve: scale down if total exceeds 1 - reserve (Requirement 5.5).
	alloc := 1.0 - cfg.Risk.CashReserve
	var sum float64
	for _, weight := range w {
		sum += weight
	}
	if sum > alloc && sum > 0 {
		scale := alloc / sum
		for sym := range w {
			w[sym] *= scale
		}
	}
	return w, nil
}
