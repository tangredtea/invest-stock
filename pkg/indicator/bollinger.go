package indicator

import "math"

// BollingerBands calculates Bollinger Bands (middle, upper, lower).
// Middle = SMA(period), Upper/Lower = Middle ± mult*StdDev.
func BollingerBands(closes []float64, period int, mult float64) (upper, middle, lower []float64) {
	n := len(closes)
	middle = SMA(closes, period)
	upper = make([]float64, n)
	lower = make([]float64, n)

	for i := period - 1; i < n; i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			d := closes[j] - middle[i]
			sum += d * d
		}
		std := math.Sqrt(sum / float64(period))
		upper[i] = middle[i] + mult*std
		lower[i] = middle[i] - mult*std
	}
	return
}
