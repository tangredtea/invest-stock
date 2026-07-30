package backtest

import "math"

func finiteFloat(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

func finiteOrZero(v float64) float64 {
	if !finiteFloat(v) {
		return 0
	}
	return v
}
