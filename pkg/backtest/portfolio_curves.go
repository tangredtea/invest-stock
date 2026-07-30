package backtest

type portfolioCurves struct {
	equity     []float64
	drawdown   []float64
	netValue   []float64
	weights    map[string][]float64
	cashWeight []float64
}

func newPortfolioCurves(symbols []string, n int) portfolioCurves {
	curves := portfolioCurves{
		equity:     make([]float64, n),
		drawdown:   make([]float64, n),
		netValue:   make([]float64, n),
		weights:    make(map[string][]float64, len(symbols)),
		cashWeight: make([]float64, n),
	}
	for _, sym := range symbols {
		curves.weights[sym] = make([]float64, n)
	}
	return curves
}

// record marks the portfolio to market at one bar and returns the updated peak.
func (c portfolioCurves) record(t int, acc *portfolioAccount, symbols []string, prices map[string]float64, peak, initialCash float64) float64 {
	eq := acc.equity(prices)
	c.equity[t] = eq
	c.netValue[t] = eq / initialCash
	if eq > peak {
		peak = eq
	}
	if peak > 0 {
		dd := (peak - eq) / peak
		if dd < 0 {
			dd = 0
		}
		c.drawdown[t] = dd
	}
	for _, sym := range symbols {
		c.weights[sym][t] = acc.weightWithEquity(sym, prices, eq)
	}
	if eq > 0 {
		c.cashWeight[t] = acc.cash / eq
	}
	return peak
}
