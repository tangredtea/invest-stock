package backtest

import (
	"fmt"
	"time"
)

// FillRule decides at which price a signal is filled (Requirement 5.3).
type FillRule int

const (
	FillNextOpen FillRule = iota // next bar's open (default, more realistic)
	FillClose                    // current bar's close
)

const (
	minInitialCash  = 0.01
	maxInitialCash  = 1e12
	defaultInitCash = 100000.0
)

// Config is the backtest configuration (Requirements 6, 8, 11.7).
type Config struct {
	InitialCash  float64    `json:"initialCash"` // [0.01, 1e12]
	StartDate    *time.Time `json:"startDate"`   // optional; nil = unbounded
	EndDate      *time.Time `json:"endDate"`     // optional
	FillRule     FillRule   `json:"fillRule"`
	Cost         CostModel  `json:"cost"`
	TPlus1       bool       `json:"tPlus1"`       // default true
	RiskFreeRate float64    `json:"riskFreeRate"` // annualized, default 0
}

// Validate checks config self-consistency (Requirements 5.9, 6.2, 6.5).
func (c Config) Validate() error {
	if !finiteFloat(c.InitialCash) || c.InitialCash < minInitialCash || c.InitialCash > maxInitialCash {
		return fmt.Errorf("初始资金越界: %g 不在 [%g, %g]", c.InitialCash, minInitialCash, maxInitialCash)
	}
	if c.FillRule != FillNextOpen && c.FillRule != FillClose {
		return fmt.Errorf("不支持的成交规则: %d", c.FillRule)
	}
	if !finiteFloat(c.RiskFreeRate) {
		return fmt.Errorf("无风险利率必须为有限数: %g", c.RiskFreeRate)
	}
	if c.StartDate != nil && c.EndDate != nil && c.StartDate.After(*c.EndDate) {
		return fmt.Errorf("回测起始日期晚于结束日期")
	}
	if err := c.Cost.validate(); err != nil {
		return err
	}
	return nil
}

// DefaultConfig returns the platform default: InitialCash=100000, FillNextOpen,
// TPlus1=true, RiskFreeRate=0, zero cost (used by compat layer and default runs).
func DefaultConfig() Config {
	return Config{
		InitialCash:  defaultInitCash,
		FillRule:     FillNextOpen,
		Cost:         ZeroCostModel(),
		TPlus1:       true,
		RiskFreeRate: 0,
	}
}
