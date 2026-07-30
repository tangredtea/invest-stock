package backtest

import (
	"fmt"
	"math"
	"time"
)

// annualizationFactor is the number of trading days used for annualization.
const annualizationFactor = 252.0

// safe returns 0 for non-finite values, otherwise v (numerical robustness).
func safe(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}

// Calculate computes all performance metrics from the equity curve and trades
// (Requirements 9–13). initialCash must be > 0 (Requirement 9.1).
func Calculate(
	equity []float64,
	dates []time.Time,
	trades []TradeRecord,
	initialCash float64,
	calendarDays int,
	riskFreeAnnual float64,
) (Metrics, error) {
	if initialCash <= 0 {
		return Metrics{}, fmt.Errorf("初始资金必须为正数: %g", initialCash)
	}
	var m Metrics

	n := len(equity)
	if n == 0 {
		return m, nil
	}

	// --- Returns (Requirement 9) ---
	m.TotalReturn, m.AnnualReturn = returnStats(equity[n-1], initialCash, calendarDays)

	// --- Drawdown (Requirement 10) ---
	m.MaxDrawdown, m.MaxDDPeakDate, m.MaxDDTroughDate, m.LongestDDBars = drawdownStats(equity, dates)

	// --- Daily simple returns ---
	rets := dailyReturns(equity)

	// --- Annualized volatility (Requirement 10.4, 10.6) ---
	mean, sampleStd := meanSampleStd(rets)
	m.AnnualVolatility = safe(sampleStd * math.Sqrt(annualizationFactor))

	// --- Sharpe (Requirements 11.1, 11.4, 11.8, 13.4, 13.5) ---
	dailyRF := riskFreeAnnual / annualizationFactor
	if len(rets) >= 2 && sampleStd > 0 {
		annualExcess := (mean - dailyRF) * annualizationFactor
		m.Sharpe = safe(annualExcess / m.AnnualVolatility)
	} else {
		m.Sharpe = 0
	}

	// --- Sortino (Requirements 11.2, 11.5, 11.8) ---
	if len(rets) >= 2 {
		downsideVol := downsideDeviation(rets, dailyRF) * math.Sqrt(annualizationFactor)
		if downsideVol > 0 {
			annualExcess := (mean - dailyRF) * annualizationFactor
			m.Sortino = safe(annualExcess / downsideVol)
		}
	}

	// --- Calmar (Requirements 11.3, 11.6) ---
	if math.Abs(m.MaxDrawdown) > 0 {
		m.Calmar = safe(m.AnnualReturn / math.Abs(m.MaxDrawdown))
	}

	// --- Trade stats (Requirement 12) ---
	applyTradeStats(&m, trades, initialCash)

	return m, nil
}
