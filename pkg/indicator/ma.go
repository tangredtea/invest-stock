package indicator

// SMA calculates Simple Moving Average for the given period.
// Returns a slice of same length as input; first period-1 values are 0.
func SMA(closes []float64, period int) []float64 {
	n := len(closes)
	out := make([]float64, n)
	if period <= 0 || n < period {
		return out
	}
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += closes[i]
	}
	out[period-1] = sum / float64(period)
	for i := period; i < n; i++ {
		sum += closes[i] - closes[i-period]
		out[i] = sum / float64(period)
	}
	return out
}
