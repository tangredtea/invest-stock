package backtest

import "invest/pkg/indicator"

// computePortfolioIndicatorSeries precomputes per-symbol indicator series on
// aligned closes (Requirement 3.3).
func computePortfolioIndicatorSeries(aligned AlignedData, symbols []string) map[string]indicator.Series {
	n := len(aligned.Timeline)
	series := make(map[string]indicator.Series, len(symbols))
	for _, sym := range symbols {
		closes := make([]float64, n)
		for i, p := range aligned.Points[sym] {
			closes[i] = p.Close
		}
		series[sym] = indicator.ComputeSeries(closes)
	}
	return series
}
