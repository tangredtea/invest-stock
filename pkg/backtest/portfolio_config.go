package backtest

import "fmt"

// WeightScheme selects how target weights are assigned (Requirements 5.6, 5.9).
type WeightScheme int

const (
	WeightEqual      WeightScheme = iota // equal weight (P0)
	WeightSpecified                      // specified weights (P0)
	WeightInverseVol                     // inverse-volatility (P2)
)

// StopLossRef selects the reference price for stop-loss (Requirement 5.1).
type StopLossRef int

const (
	StopLossByCost StopLossRef = iota // position cost basis
	StopLossByPeak                    // highest close since entry
)

// RebalanceConfig configures rebalancing (Requirement 4).
type RebalanceConfig struct {
	Periodic       bool    `json:"periodic"`
	PeriodBars     int     `json:"periodBars"`     // N >= 1 when Periodic
	Threshold      bool    `json:"threshold"`
	ThresholdValue float64 `json:"thresholdValue"` // (0,1] when Threshold
}

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

// PortfolioConfig is the portfolio backtest configuration (Requirement 3.11).
// It embeds the existing single-symbol Config for cost/fill/cash/date semantics.
type PortfolioConfig struct {
	Base             Config             `json:"base"`
	Policy           AlignmentPolicy    `json:"policy"`
	Scheme           WeightScheme       `json:"scheme"`
	Weights          map[string]float64 `json:"weights"` // specified weights
	InverseVolWindow int                `json:"inverseVolWindow"`
	Rebalance        RebalanceConfig    `json:"rebalance"`
	Risk             RiskConfig         `json:"risk"`
	Correlation      bool               `json:"correlation"` // P2
}

const weightSumTolerance = 1e-9

// Validate checks the portfolio config for self-consistency, returning the
// first failing item as a descriptive error (Requirements 3.11, 4.6, 5.8, 5.10).
func (c PortfolioConfig) Validate() error {
	// 1. Base config (cash/date/cost bounds).
	if err := c.Base.Validate(); err != nil {
		return err
	}
	// 2. Alignment policy.
	if c.Policy != AlignIntersection && c.Policy != AlignUnionFFill {
		return fmt.Errorf("不支持的对齐策略: %d", c.Policy)
	}
	// 3. Rebalance config (Requirement 4.6).
	if c.Rebalance.Periodic && c.Rebalance.PeriodBars < 1 {
		return fmt.Errorf("再平衡周期必须为正整数, 实际 %d", c.Rebalance.PeriodBars)
	}
	if c.Rebalance.Threshold && (c.Rebalance.ThresholdValue <= 0 || c.Rebalance.ThresholdValue > 1) {
		return fmt.Errorf("再平衡阈值必须在 (0,1], 实际 %g", c.Rebalance.ThresholdValue)
	}
	// 4. Weight scheme (Requirement 5.8).
	switch c.Scheme {
	case WeightSpecified:
		var sum float64
		for sym, w := range c.Weights {
			if w < 0 {
				return fmt.Errorf("标的 %s 的指定权重为负: %g", sym, w)
			}
			sum += w
		}
		if sum > 1+weightSumTolerance {
			return fmt.Errorf("指定权重之和 %g 超过 1", sum)
		}
	case WeightInverseVol:
		if c.InverseVolWindow < 1 {
			return fmt.Errorf("波动率反比回溯窗口必须为正整数, 实际 %d", c.InverseVolWindow)
		}
	}
	// 5. Risk config (Requirements 5.8, 5.10).
	if err := c.Risk.validate(); err != nil {
		return err
	}
	return nil
}

func (r RiskConfig) validate() error {
	if r.StopLossEnabled && (r.StopLossPct <= 0 || r.StopLossPct > 1) {
		return fmt.Errorf("止损阈值必须在 (0,1], 实际 %g", r.StopLossPct)
	}
	if r.TakeProfitEnabled && r.TakeProfitPct <= 0 {
		return fmt.Errorf("止盈阈值必须为正数, 实际 %g", r.TakeProfitPct)
	}
	if r.MaxDDGuardEnabled {
		if r.MaxDDThreshold <= 0 || r.MaxDDThreshold > 1 {
			return fmt.Errorf("最大回撤阈值必须在 (0,1], 实际 %g", r.MaxDDThreshold)
		}
		if r.MaxDDTargetExposure < 0 || r.MaxDDTargetExposure > 1 {
			return fmt.Errorf("回撤触发后目标持仓比例必须在 [0,1], 实际 %g", r.MaxDDTargetExposure)
		}
	}
	for sym, cap := range r.PerSymbolCap {
		if cap < 0 || cap > 1 {
			return fmt.Errorf("标的 %s 的权重上限必须在 [0,1], 实际 %g", sym, cap)
		}
	}
	if r.CashReserve < 0 || r.CashReserve > 1 {
		return fmt.Errorf("现金保留比例必须在 [0,1], 实际 %g", r.CashReserve)
	}
	return nil
}

// DefaultPortfolioConfig returns a sensible default: equal weight, intersection
// alignment, default base config, no rebalance, no risk constraints.
func DefaultPortfolioConfig() PortfolioConfig {
	return PortfolioConfig{
		Base:   DefaultConfig(),
		Policy: AlignIntersection,
		Scheme: WeightEqual,
	}
}
