package backtest

import "invest/pkg/indicator"

// schemeWeights computes target weights per the configured WeightScheme over
// the tradable symbols (Requirements 5.6, 5.7, 5.9).
func (e PortfolioEngine) schemeWeights(cfg PortfolioConfig, tradable []string, aligned AlignedData, series map[string]indicator.Series, t int) map[string]float64 {
	w := make(map[string]float64, len(tradable))
	alloc := 1.0 - cfg.Risk.CashReserve
	switch cfg.Scheme {
	case WeightSpecified:
		for _, sym := range tradable {
			w[sym] = cfg.Weights[sym]
		}
	case WeightInverseVol:
		invVol := make(map[string]float64, len(tradable))
		var total float64
		for _, sym := range tradable {
			v := e.recentVol(aligned, sym, t, cfg.InverseVolWindow)
			if v > 0 {
				invVol[sym] = 1 / v
			}
			total += invVol[sym]
		}
		if total > 0 {
			for _, sym := range tradable {
				w[sym] = alloc * invVol[sym] / total
			}
		} else {
			w = equalWeights(tradable, alloc)
		}
	default: // WeightEqual
		w = equalWeights(tradable, alloc)
	}
	return w
}

// equalWeights assigns an equal share of alloc to each symbol (Requirement 5.7).
func equalWeights(syms []string, alloc float64) map[string]float64 {
	w := make(map[string]float64, len(syms))
	if len(syms) == 0 {
		return w
	}
	each := alloc / float64(len(syms))
	for _, sym := range syms {
		w[sym] = each
	}
	return w
}

// recentVol returns the sample std of recent daily returns for a symbol.
func (e PortfolioEngine) recentVol(aligned AlignedData, sym string, t, window int) float64 {
	pts := aligned.Points[sym]
	start := t - window + 1
	if start < 1 {
		start = 1
	}
	var rets []float64
	for i := start; i <= t; i++ {
		if pts[i-1].Close > 0 {
			rets = append(rets, pts[i].Close/pts[i-1].Close-1)
		}
	}
	_, std := meanSampleStd(rets)
	return std
}
