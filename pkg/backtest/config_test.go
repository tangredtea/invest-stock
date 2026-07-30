package backtest

import (
	"math"
	"testing"
)

func TestConfigValidateRejectsNonFiniteCostParams(t *testing.T) {
	cases := []struct {
		name string
		edit func(*Config)
	}{
		{"initial cash Inf", func(c *Config) { c.InitialCash = math.Inf(1) }},
		{"risk-free rate NaN", func(c *Config) { c.RiskFreeRate = math.NaN() }},
		{"invalid fill rule", func(c *Config) { c.FillRule = FillRule(99) }},
		{"invalid slippage mode", func(c *Config) { c.Cost.SlippageMode = SlippageMode(99) }},
		{"commission rate NaN", func(c *Config) { c.Cost.CommissionRate = math.NaN() }},
		{"min commission Inf", func(c *Config) { c.Cost.MinCommission = math.Inf(1) }},
		{"stamp tax rate NaN", func(c *Config) { c.Cost.StampTaxRate = math.NaN() }},
		{"slippage ratio Inf", func(c *Config) { c.Cost.SlippageRatio = math.Inf(1) }},
		{"tick NaN in tick mode", func(c *Config) {
			c.Cost.SlippageMode = SlippageTick
			c.Cost.SlippageTicks = 1
			c.Cost.Tick = math.NaN()
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tc.edit(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestConfigValidateAcceptsTickSlippage(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Cost.SlippageMode = SlippageTick
	cfg.Cost.Tick = 0.01
	cfg.Cost.SlippageTicks = 2
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate rejected tick slippage: %v", err)
	}
}
