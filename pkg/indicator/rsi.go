package indicator

import "math"

// RSI calculates the Relative Strength Index for the given period.
// Returns a slice of same length as input; first period values are 0.
func RSI(closes []float64, period int) []float64 {
	n := len(closes)
	out := make([]float64, n)
	if period <= 0 || n <= period {
		return out
	}

	gainSum, lossSum := 0.0, 0.0
	for i := 1; i <= period; i++ {
		d := closes[i] - closes[i-1]
		if d > 0 {
			gainSum += d
		} else {
			lossSum += math.Abs(d)
		}
	}
	avgGain := gainSum / float64(period)
	avgLoss := lossSum / float64(period)

	if avgLoss == 0 {
		out[period] = 100
	} else {
		out[period] = 100 - 100/(1+avgGain/avgLoss)
	}

	for i := period + 1; i < n; i++ {
		d := closes[i] - closes[i-1]
		gain, loss := 0.0, 0.0
		if d > 0 {
			gain = d
		} else {
			loss = math.Abs(d)
		}
		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)
		if avgLoss == 0 {
			out[i] = 100
		} else {
			out[i] = 100 - 100/(1+avgGain/avgLoss)
		}
	}
	return out
}
