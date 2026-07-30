package backtest

import "fmt"

// WeightScheme selects how target weights are assigned (Requirements 5.6, 5.9).
type WeightScheme int

const (
	WeightEqual      WeightScheme = iota // equal weight (P0)
	WeightSpecified                      // specified weights (P0)
	WeightInverseVol                     // inverse-volatility (P2)
)

// RebalanceConfig configures rebalancing (Requirement 4).
type RebalanceConfig struct {
	Periodic       bool    `json:"periodic"`
	PeriodBars     int     `json:"periodBars"` // N >= 1 when Periodic
	Threshold      bool    `json:"threshold"`
	ThresholdValue float64 `json:"thresholdValue"` // (0,1] when Threshold
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
	if c.Rebalance.Threshold && (!finiteFloat(c.Rebalance.ThresholdValue) || c.Rebalance.ThresholdValue <= 0 || c.Rebalance.ThresholdValue > 1) {
		return fmt.Errorf("再平衡阈值必须在 (0,1], 实际 %g", c.Rebalance.ThresholdValue)
	}
	// 4. Weight scheme (Requirement 5.8).
	switch c.Scheme {
	case WeightEqual:
	case WeightSpecified:
		var sum float64
		for sym, w := range c.Weights {
			if !finiteFloat(w) || w < 0 || w > 1 {
				return fmt.Errorf("标的 %s 的指定权重必须在 [0,1], 实际 %g", sym, w)
			}
			sum += w
		}
		if !finiteFloat(sum) {
			return fmt.Errorf("指定权重之和不是有限数: %g", sum)
		}
		if sum > 1+weightSumTolerance {
			return fmt.Errorf("指定权重之和 %g 超过 1", sum)
		}
	case WeightInverseVol:
		if c.InverseVolWindow < 1 {
			return fmt.Errorf("波动率反比回溯窗口必须为正整数, 实际 %d", c.InverseVolWindow)
		}
	default:
		return fmt.Errorf("不支持的权重方案: %d", c.Scheme)
	}
	// 5. Risk config (Requirements 5.8, 5.10).
	if err := c.Risk.validate(); err != nil {
		return err
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
