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

	final := equity[n-1]

	// --- Returns (Requirement 9) ---
	m.TotalReturn = safe((final - initialCash) / initialCash)
	if calendarDays > 0 && 1+m.TotalReturn > 0 {
		years := 365.0 / float64(calendarDays)
		m.AnnualReturn = safe(math.Pow(1+m.TotalReturn, years) - 1)
	} else {
		m.AnnualReturn = 0
	}

	// --- Drawdown (Requirement 10) ---
	m.MaxDrawdown, m.MaxDDPeakDate, m.MaxDDTroughDate, m.LongestDDBars = drawdownStats(equity, dates)

	// --- Daily simple returns ---
	rets := make([]float64, 0, n-1)
	for i := 1; i < n; i++ {
		if equity[i-1] != 0 {
			rets = append(rets, equity[i]/equity[i-1]-1)
		} else {
			rets = append(rets, 0)
		}
	}

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
	m.TotalTrades = len(trades)
	winSum, lossSum := 0.0, 0.0
	winN, lossN, closedN := 0, 0, 0
	var turnoverSum float64
	for _, t := range trades {
		turnoverSum += t.Turnover
		if t.Action == "sell" {
			closedN++
			if t.RealizedPL > 0 {
				winSum += t.RealizedPL
				winN++
			} else {
				lossSum += t.RealizedPL
				lossN++
			}
		}
	}
	if closedN > 0 {
		m.WinRate = safe(float64(winN) / float64(closedN))
	}
	if winN > 0 {
		m.AvgWin = safe(winSum / float64(winN))
	}
	if lossN > 0 {
		m.AvgLoss = safe(lossSum / float64(lossN))
	}
	if lossN > 0 && winN > 0 && m.AvgLoss != 0 {
		m.ProfitLossRatio = safe(m.AvgWin / math.Abs(m.AvgLoss))
	}
	m.Turnover = safe(turnoverSum / initialCash)

	return m, nil
}

// drawdownStats computes max drawdown (in [0,1]), its peak/trough dates (earliest
// on ties), and the longest drawdown duration in bars (Requirements 10.1, 10.2,
// 10.3, 10.7, 10.8).
func drawdownStats(equity []float64, dates []time.Time) (maxDD float64, peakDate, troughDate time.Time, longestBars int) {
	n := len(equity)
	if n == 0 {
		return 0, time.Time{}, time.Time{}, 0
	}
	if len(dates) > 0 {
		peakDate = dates[0]
		troughDate = dates[0]
	}

	peak := equity[0]
	peakIdx := 0
	curDDStart := 0 // index where current drawdown began (last new high)

	for i := 0; i < n; i++ {
		if equity[i] > peak {
			peak = equity[i]
			peakIdx = i
			curDDStart = i
		}
		var dd float64
		if peak > 0 {
			dd = (peak - equity[i]) / peak
		}
		if dd > maxDD {
			maxDD = dd
			if len(dates) > 0 {
				peakDate = dates[peakIdx]
				troughDate = dates[i]
			}
		}
		// Track longest drawdown duration: bars since last new high.
		if dur := i - curDDStart; dur > longestBars {
			longestBars = dur
		}
	}
	if maxDD < 0 {
		maxDD = 0
	}
	if maxDD > 1 {
		maxDD = 1
	}
	return maxDD, peakDate, troughDate, longestBars
}

// meanSampleStd returns the mean and the sample standard deviation (denominator
// N-1) of xs. For len < 2 the std is 0.
func meanSampleStd(xs []float64) (mean, std float64) {
	n := len(xs)
	if n == 0 {
		return 0, 0
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	mean = sum / float64(n)
	if n < 2 {
		return mean, 0
	}
	var sq float64
	for _, x := range xs {
		d := x - mean
		sq += d * d
	}
	variance := sq / float64(n-1)
	if variance <= 0 {
		return mean, 0
	}
	return mean, math.Sqrt(variance)
}

// downsideDeviation returns sqrt( sum( min(0, r - target)^2 ) / N ), the
// downside deviation of returns below target (Requirement 11.2).
func downsideDeviation(xs []float64, target float64) float64 {
	n := len(xs)
	if n == 0 {
		return 0
	}
	var sq float64
	for _, x := range xs {
		if x < target {
			d := x - target
			sq += d * d
		}
	}
	return math.Sqrt(sq / float64(n))
}
