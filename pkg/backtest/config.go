package backtest

import (
	"fmt"
	"math"
	"time"
)

// FillRule decides at which price a signal is filled (Requirement 5.3).
type FillRule int

const (
	FillNextOpen FillRule = iota // next bar's open (default, more realistic)
	FillClose                    // current bar's close
)

// SlippageMode selects how slippage is applied (Requirement 7.4).
type SlippageMode int

const (
	SlippageRatio SlippageMode = iota // proportional
	SlippageTick                      // integer multiples of the tick size
)

const (
	maxCommissionRate = 0.01
	maxMinCommission  = 1000.0
	maxStampTaxRate   = 0.01
	maxSlippageRatio  = 0.1
	minInitialCash    = 0.01
	maxInitialCash    = 1e12
	defaultInitCash   = 100000.0
	defaultStampTax   = 0.0005
	defaultTick       = 0.01
)

// CostModel models commission, stamp tax, and slippage (Requirement 7).
type CostModel struct {
	CommissionRate float64      `json:"commissionRate"` // [0, 0.01]
	MinCommission  float64      `json:"minCommission"`  // [0, 1000]
	StampTaxRate   float64      `json:"stampTaxRate"`   // sell only, default 0.0005, [0, 0.01]
	SlippageMode   SlippageMode `json:"slippageMode"`
	SlippageRatio  float64      `json:"slippageRatio"` // [0, 0.1]
	Tick           float64      `json:"tick"`          // > 0, default 0.01
	SlippageTicks  int          `json:"slippageTicks"` // slippage = SlippageTicks * Tick
}

// FillPrice applies slippage to the base price in the unfavorable direction
// (buy >= base, sell <= base) (Requirements 7.4, 7.5).
func (m CostModel) FillPrice(base float64, action Action) float64 {
	var slip float64
	switch m.SlippageMode {
	case SlippageTick:
		slip = float64(m.SlippageTicks) * m.Tick
	default: // SlippageRatio
		slip = base * m.SlippageRatio
	}
	switch action {
	case Buy:
		return base + slip
	case Sell:
		p := base - slip
		if p < 0 {
			p = 0
		}
		return p
	default:
		return base
	}
}

// Commission computes the fee for a turnover, applying the floor (Requirements 7.1, 7.2).
func (m CostModel) Commission(turnover float64) float64 {
	fee := turnover * m.CommissionRate
	if fee < m.MinCommission {
		return m.MinCommission
	}
	return fee
}

// StampTax is charged on sells only (Requirement 7.3).
func (m CostModel) StampTax(turnover float64, action Action) float64 {
	if action != Sell {
		return 0
	}
	return turnover * m.StampTaxRate
}

// ZeroCostModel returns a cost model that applies no costs.
func ZeroCostModel() CostModel {
	return CostModel{Tick: defaultTick}
}

// validate checks the cost model parameter ranges (Requirements 7.1, 7.3, 7.4).
func (m CostModel) validate() error {
	if m.CommissionRate < 0 || m.CommissionRate > maxCommissionRate {
		return fmt.Errorf("手续费比例越界: %g 不在 [0, %g]", m.CommissionRate, maxCommissionRate)
	}
	if m.MinCommission < 0 || m.MinCommission > maxMinCommission {
		return fmt.Errorf("最低手续费越界: %g 不在 [0, %g]", m.MinCommission, maxMinCommission)
	}
	if m.StampTaxRate < 0 || m.StampTaxRate > maxStampTaxRate {
		return fmt.Errorf("印花税率越界: %g 不在 [0, %g]", m.StampTaxRate, maxStampTaxRate)
	}
	if m.SlippageRatio < 0 || m.SlippageRatio > maxSlippageRatio {
		return fmt.Errorf("滑点比例越界: %g 不在 [0, %g]", m.SlippageRatio, maxSlippageRatio)
	}
	if m.SlippageMode == SlippageTick && m.Tick <= 0 {
		return fmt.Errorf("最小变动价位必须为正数: %g", m.Tick)
	}
	if m.SlippageTicks < 0 {
		return fmt.Errorf("滑点 tick 数不能为负: %d", m.SlippageTicks)
	}
	return nil
}

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
	if math.IsNaN(c.InitialCash) || c.InitialCash < minInitialCash || c.InitialCash > maxInitialCash {
		return fmt.Errorf("初始资金越界: %g 不在 [%g, %g]", c.InitialCash, minInitialCash, maxInitialCash)
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
