package backtest

import "math"

func returnStats(final, initialCash float64, calendarDays int) (totalReturn, annualReturn float64) {
	totalReturn = safe((final - initialCash) / initialCash)
	if calendarDays > 0 && 1+totalReturn > 0 {
		years := 365.0 / float64(calendarDays)
		annualReturn = safe(math.Pow(1+totalReturn, years) - 1)
	}
	return totalReturn, annualReturn
}

func dailyReturns(equity []float64) []float64 {
	rets := make([]float64, 0, len(equity)-1)
	for i := 1; i < len(equity); i++ {
		if equity[i-1] != 0 {
			rets = append(rets, equity[i]/equity[i-1]-1)
		} else {
			rets = append(rets, 0)
		}
	}
	return rets
}
