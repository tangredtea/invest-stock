package main

import "invest/pkg/backtest"

// toPortfolioConfig builds a backtest.PortfolioConfig from the request.
func (req portfolioReq) toPortfolioConfig() (backtest.PortfolioConfig, error) {
	cfg := backtest.DefaultPortfolioConfig()
	if req.Config != nil {
		base, err := req.Config.toConfig()
		if err != nil {
			return cfg, err
		}
		cfg.Base = base
	}
	cfg.Policy = backtest.AlignmentPolicy(req.Policy)
	cfg.Scheme = backtest.WeightScheme(req.Scheme)
	cfg.Weights = req.Weights
	if req.Rebalance != nil {
		cfg.Rebalance = backtest.RebalanceConfig{
			Periodic:       req.Rebalance.Periodic,
			PeriodBars:     req.Rebalance.PeriodBars,
			Threshold:      req.Rebalance.Threshold,
			ThresholdValue: req.Rebalance.ThresholdValue,
		}
	}
	if req.Risk != nil {
		cfg.Risk = backtest.RiskConfig{
			StopLossEnabled:     req.Risk.StopLossEnabled,
			StopLossPct:         req.Risk.StopLossPct,
			StopLossRef:         req.stopLossRef(),
			TakeProfitEnabled:   req.Risk.TakeProfitEnabled,
			TakeProfitPct:       req.Risk.TakeProfitPct,
			MaxDDGuardEnabled:   req.Risk.MaxDDGuardEnabled,
			MaxDDThreshold:      req.Risk.MaxDDThreshold,
			MaxDDTargetExposure: req.Risk.MaxDDTargetExposure,
			PerSymbolCap:        req.Risk.PerSymbolCap,
			CashReserve:         req.Risk.CashReserve,
		}
	}
	return cfg, nil
}

func (req portfolioReq) stopLossRef() backtest.StopLossRef {
	if req.Risk != nil && req.Risk.StopLossByPeak {
		return backtest.StopLossByPeak
	}
	return backtest.StopLossByCost
}

// buildStrategy builds either a per-symbol adapter (wrapping a builtin) or the
// scheme-weights portfolio strategy.
func (req portfolioReq) buildStrategy(symbols []string) (backtest.PortfolioStrategy, error) {
	if req.PerSymbol {
		return backtest.NewPerSymbolAdapter(backtest.Default, req.Strategy, req.Params, symbols)
	}
	return backtest.SchemeStrategy{}, nil
}
