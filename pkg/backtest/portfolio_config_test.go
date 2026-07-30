package backtest

import (
	"math"
	"testing"
)

func TestPortfolioConfigValidateRejectsNonFiniteFloats(t *testing.T) {
	cases := []struct {
		name string
		edit func(*PortfolioConfig)
	}{
		{"rebalance threshold NaN", func(c *PortfolioConfig) {
			c.Rebalance.Threshold = true
			c.Rebalance.ThresholdValue = math.NaN()
		}},
		{"specified weight Inf", func(c *PortfolioConfig) {
			c.Scheme = WeightSpecified
			c.Weights = map[string]float64{"600000": math.Inf(1)}
		}},
		{"stop loss NaN", func(c *PortfolioConfig) {
			c.Risk.StopLossEnabled = true
			c.Risk.StopLossPct = math.NaN()
		}},
		{"take profit Inf", func(c *PortfolioConfig) {
			c.Risk.TakeProfitEnabled = true
			c.Risk.TakeProfitPct = math.Inf(1)
		}},
		{"max drawdown threshold NaN", func(c *PortfolioConfig) {
			c.Risk.MaxDDGuardEnabled = true
			c.Risk.MaxDDThreshold = math.NaN()
			c.Risk.MaxDDTargetExposure = 0.5
		}},
		{"max drawdown exposure Inf", func(c *PortfolioConfig) {
			c.Risk.MaxDDGuardEnabled = true
			c.Risk.MaxDDThreshold = 0.2
			c.Risk.MaxDDTargetExposure = math.Inf(1)
		}},
		{"per symbol cap NaN", func(c *PortfolioConfig) {
			c.Risk.PerSymbolCap = map[string]float64{"600000": math.NaN()}
		}},
		{"cash reserve Inf", func(c *PortfolioConfig) {
			c.Risk.CashReserve = math.Inf(1)
		}},
		{"invalid weight scheme", func(c *PortfolioConfig) {
			c.Scheme = WeightScheme(99)
		}},
		{"invalid stop loss ref", func(c *PortfolioConfig) {
			c.Risk.StopLossRef = StopLossRef(99)
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultPortfolioConfig()
			tc.edit(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestPortfolioConfigValidateRejectsSpecifiedWeightOverOne(t *testing.T) {
	cfg := DefaultPortfolioConfig()
	cfg.Scheme = WeightSpecified
	cfg.Weights = map[string]float64{"600000": 1.1}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}
