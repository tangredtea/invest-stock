package main

import (
	"time"

	"invest/pkg/backtest"
)

// backtestReq is one backtest request: a security code, a strategy name +
// params, and an optional config override.
type backtestReq struct {
	Code     string                         `json:"code"`
	Strategy string                         `json:"strategy"`
	Params   map[string]backtest.ParamValue `json:"params"`
	Config   *backtestConfigReq             `json:"config"`
}

type backtestCompareReq struct {
	Code       string               `json:"code"`
	Config     *backtestConfigReq   `json:"config"`
	Strategies []compareStrategyReq `json:"strategies"`
}

type compareStrategyReq struct {
	Strategy string                         `json:"strategy"`
	Params   map[string]backtest.ParamValue `json:"params"`
}

// backtestConfigReq mirrors the configurable subset of backtest.Config in JSON.
type backtestConfigReq struct {
	InitialCash    *float64 `json:"initialCash"`
	StartDate      string   `json:"startDate"` // "2006-01-02", optional
	EndDate        string   `json:"endDate"`
	FillRule       *int     `json:"fillRule"`
	TPlus1         *bool    `json:"tPlus1"`
	RiskFreeRate   *float64 `json:"riskFreeRate"`
	CommissionRate *float64 `json:"commissionRate"`
	MinCommission  *float64 `json:"minCommission"`
	StampTaxRate   *float64 `json:"stampTaxRate"`
	SlippageMode   *int     `json:"slippageMode"`
	SlippageRatio  *float64 `json:"slippageRatio"`
	Tick           *float64 `json:"tick"`
	SlippageTicks  *int     `json:"slippageTicks"`
}

// toConfig builds a backtest.Config from the request, starting from defaults.
func (c *backtestConfigReq) toConfig() (backtest.Config, error) {
	cfg := backtest.DefaultConfig()
	if c == nil {
		return cfg, nil
	}
	if c.InitialCash != nil {
		cfg.InitialCash = *c.InitialCash
	}
	if c.FillRule != nil {
		cfg.FillRule = backtest.FillRule(*c.FillRule)
	}
	if c.TPlus1 != nil {
		cfg.TPlus1 = *c.TPlus1
	}
	if c.RiskFreeRate != nil {
		cfg.RiskFreeRate = *c.RiskFreeRate
	}
	if c.CommissionRate != nil {
		cfg.Cost.CommissionRate = *c.CommissionRate
	}
	if c.MinCommission != nil {
		cfg.Cost.MinCommission = *c.MinCommission
	}
	if c.StampTaxRate != nil {
		cfg.Cost.StampTaxRate = *c.StampTaxRate
	}
	if c.SlippageMode != nil {
		cfg.Cost.SlippageMode = backtest.SlippageMode(*c.SlippageMode)
	}
	if c.SlippageRatio != nil {
		cfg.Cost.SlippageRatio = *c.SlippageRatio
	}
	if c.Tick != nil {
		cfg.Cost.Tick = *c.Tick
	}
	if c.SlippageTicks != nil {
		cfg.Cost.SlippageTicks = *c.SlippageTicks
	}
	if c.StartDate != "" {
		t, err := time.Parse("2006-01-02", c.StartDate)
		if err != nil {
			return cfg, errBadDate
		}
		cfg.StartDate = &t
	}
	if c.EndDate != "" {
		t, err := time.Parse("2006-01-02", c.EndDate)
		if err != nil {
			return cfg, errBadDate
		}
		cfg.EndDate = &t
	}
	return cfg, nil
}

var errBadDate = &apiError{"日期格式无效, 应为 2006-01-02"}

type apiError struct{ msg string }

func (e *apiError) Error() string { return e.msg }
