package backtest

import "time"

// drawdownStats computes max drawdown (in [0,1]), its peak/trough dates
// (earliest on ties), and the longest drawdown duration in bars (Requirements
// 10.1, 10.2, 10.3, 10.7, 10.8).
func drawdownStats(equity []float64, dates []time.Time) (maxDD float64, peakDate, troughDate time.Time, longestBars int) {
	n := len(equity)
	if n == 0 {
		return 0, time.Time{}, time.Time{}, 0
	}
	hasDates := len(dates) == n
	if hasDates {
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
			if hasDates {
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
