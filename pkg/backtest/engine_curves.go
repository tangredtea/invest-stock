package backtest

import (
	"time"

	"invest/pkg/data"
)

type engineCurves struct {
	equity   []float64
	drawdown []float64
	netValue []float64
	dates    []time.Time
}

func newEngineCurves(n int) engineCurves {
	return engineCurves{
		equity:   make([]float64, n),
		drawdown: make([]float64, n),
		netValue: make([]float64, n),
		dates:    make([]time.Time, n),
	}
}

// record marks the account to market at one bar and returns the updated peak.
func (c engineCurves) record(i int, k data.KLine, acc *account, peak, initialCash float64) float64 {
	eq := acc.equity(k.Close)
	c.equity[i] = eq
	c.dates[i] = k.Date
	c.netValue[i] = eq / initialCash
	if eq > peak {
		peak = eq
	}
	if peak > 0 {
		c.drawdown[i] = (peak - eq) / peak
		if c.drawdown[i] < 0 {
			c.drawdown[i] = 0
		}
	}
	return peak
}
