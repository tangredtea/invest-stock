package backtest

import "fmt"

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
	if !finiteFloat(base) || base <= 0 {
		return 0
	}
	var slip float64
	switch m.SlippageMode {
	case SlippageTick:
		slip = float64(m.SlippageTicks) * m.Tick
	default: // SlippageRatio
		slip = base * m.SlippageRatio
	}
	if !finiteFloat(slip) {
		return 0
	}
	switch action {
	case Buy:
		return finiteOrZero(base + slip)
	case Sell:
		p := base - slip
		if p < 0 {
			p = 0
		}
		return finiteOrZero(p)
	default:
		return base
	}
}

// Commission computes the fee for a turnover, applying the floor (Requirements 7.1, 7.2).
func (m CostModel) Commission(turnover float64) float64 {
	if !finiteFloat(turnover) || turnover < 0 {
		return 0
	}
	fee := turnover * m.CommissionRate
	if !finiteFloat(fee) {
		return 0
	}
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
	if !finiteFloat(turnover) || turnover < 0 {
		return 0
	}
	tax := turnover * m.StampTaxRate
	if !finiteFloat(tax) {
		return 0
	}
	return tax
}

// ZeroCostModel returns a cost model that applies no costs.
func ZeroCostModel() CostModel {
	return CostModel{Tick: defaultTick}
}

// validate checks the cost model parameter ranges (Requirements 7.1, 7.3, 7.4).
func (m CostModel) validate() error {
	if m.SlippageMode != SlippageRatio && m.SlippageMode != SlippageTick {
		return fmt.Errorf("不支持的滑点模式: %d", m.SlippageMode)
	}
	if !finiteFloat(m.CommissionRate) || m.CommissionRate < 0 || m.CommissionRate > maxCommissionRate {
		return fmt.Errorf("手续费比例越界: %g 不在 [0, %g]", m.CommissionRate, maxCommissionRate)
	}
	if !finiteFloat(m.MinCommission) || m.MinCommission < 0 || m.MinCommission > maxMinCommission {
		return fmt.Errorf("最低手续费越界: %g 不在 [0, %g]", m.MinCommission, maxMinCommission)
	}
	if !finiteFloat(m.StampTaxRate) || m.StampTaxRate < 0 || m.StampTaxRate > maxStampTaxRate {
		return fmt.Errorf("印花税率越界: %g 不在 [0, %g]", m.StampTaxRate, maxStampTaxRate)
	}
	if !finiteFloat(m.SlippageRatio) || m.SlippageRatio < 0 || m.SlippageRatio > maxSlippageRatio {
		return fmt.Errorf("滑点比例越界: %g 不在 [0, %g]", m.SlippageRatio, maxSlippageRatio)
	}
	if m.SlippageMode == SlippageTick && (!finiteFloat(m.Tick) || m.Tick <= 0) {
		return fmt.Errorf("最小变动价位必须为正数: %g", m.Tick)
	}
	if m.SlippageTicks < 0 {
		return fmt.Errorf("滑点 tick 数不能为负: %d", m.SlippageTicks)
	}
	return nil
}
