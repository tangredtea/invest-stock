package backtest

import "math"

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
