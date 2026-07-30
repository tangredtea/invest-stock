package backtest

import "fmt"

// StopLossRef selects the reference price for stop-loss (Requirement 5.1).
type StopLossRef int

const (
	StopLossByCost StopLossRef = iota // position cost basis
	StopLossByPeak                    // highest close since entry
)

// RiskConfig configures risk and position management (Requirement 5).
type RiskConfig struct {
	StopLossEnabled bool        `json:"stopLossEnabled"`
	StopLossPct     float64     `json:"stopLossPct"` // (0,1]
	StopLossRef     StopLossRef `json:"stopLossRef"`

	TakeProfitEnabled bool    `json:"takeProfitEnabled"`
	TakeProfitPct     float64 `json:"takeProfitPct"` // > 0

	MaxDDGuardEnabled   bool    `json:"maxDDGuardEnabled"`
	MaxDDThreshold      float64 `json:"maxDDThreshold"`      // (0,1]
	MaxDDTargetExposure float64 `json:"maxDDTargetExposure"` // [0,1], 0 = liquidate

	PerSymbolCap map[string]float64 `json:"perSymbolCap"` // each in [0,1]
	CashReserve  float64            `json:"cashReserve"`  // [0,1]
}

func (r RiskConfig) validate() error {
	if r.StopLossRef != StopLossByCost && r.StopLossRef != StopLossByPeak {
		return fmt.Errorf("不支持的止损参考价: %d", r.StopLossRef)
	}
	if r.StopLossEnabled && (!finiteFloat(r.StopLossPct) || r.StopLossPct <= 0 || r.StopLossPct > 1) {
		return fmt.Errorf("止损阈值必须在 (0,1], 实际 %g", r.StopLossPct)
	}
	if r.TakeProfitEnabled && (!finiteFloat(r.TakeProfitPct) || r.TakeProfitPct <= 0) {
		return fmt.Errorf("止盈阈值必须为正数, 实际 %g", r.TakeProfitPct)
	}
	if r.MaxDDGuardEnabled {
		if !finiteFloat(r.MaxDDThreshold) || r.MaxDDThreshold <= 0 || r.MaxDDThreshold > 1 {
			return fmt.Errorf("最大回撤阈值必须在 (0,1], 实际 %g", r.MaxDDThreshold)
		}
		if !finiteFloat(r.MaxDDTargetExposure) || r.MaxDDTargetExposure < 0 || r.MaxDDTargetExposure > 1 {
			return fmt.Errorf("回撤触发后目标持仓比例必须在 [0,1], 实际 %g", r.MaxDDTargetExposure)
		}
	}
	for sym, cap := range r.PerSymbolCap {
		if !finiteFloat(cap) || cap < 0 || cap > 1 {
			return fmt.Errorf("标的 %s 的权重上限必须在 [0,1], 实际 %g", sym, cap)
		}
	}
	if !finiteFloat(r.CashReserve) || r.CashReserve < 0 || r.CashReserve > 1 {
		return fmt.Errorf("现金保留比例必须在 [0,1], 实际 %g", r.CashReserve)
	}
	return nil
}
