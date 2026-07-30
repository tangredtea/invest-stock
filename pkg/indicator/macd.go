package indicator

// EMA calculates Exponential Moving Average.
func EMA(closes []float64, period int) []float64 {
	n := len(closes)
	out := make([]float64, n)
	if period <= 0 || n == 0 {
		return out
	}
	k := 2.0 / float64(period+1)
	out[0] = closes[0]
	for i := 1; i < n; i++ {
		out[i] = closes[i]*k + out[i-1]*(1-k)
	}
	return out
}

// MACD returns macd line, signal line, and histogram.
func MACD(closes []float64, fast, slow, signal int) (macdLine, signalLine, hist []float64) {
	n := len(closes)
	if fast <= 0 || slow <= 0 || signal <= 0 {
		return make([]float64, n), make([]float64, n), make([]float64, n)
	}
	emaFast := EMA(closes, fast)
	emaSlow := EMA(closes, slow)
	macdLine = make([]float64, n)
	for i := range macdLine {
		macdLine[i] = emaFast[i] - emaSlow[i]
	}
	signalLine = EMA(macdLine, signal)
	hist = make([]float64, n)
	for i := range hist {
		hist[i] = macdLine[i] - signalLine[i]
	}
	return
}
