package backtest

import "math"

// computeAttribution computes each symbol's contribution to total portfolio
// return and its final weight (Requirements 6.3, 6.4).
//
// A symbol's contribution = (cumulative realized P/L + end-of-backtest unrealized
// P/L) / initial cash. The cash contribution is the residual that makes the sum
// equal total portfolio return, so the aggregation invariant holds exactly.
func computeAttribution(acc *portfolioAccount, symbols []string, finalCloses map[string]float64, initialCash float64) []Attribution {
	equity := acc.equity(finalCloses)
	out := make([]Attribution, 0, len(symbols))
	for _, sym := range symbols {
		realized := 0.0
		for _, tr := range acc.trades {
			if tr.Symbol == sym && tr.Action == "sell" {
				realized += tr.RealizedPL
			}
		}
		// Unrealized P/L = current market value - remaining cost basis.
		sh := acc.shares(sym)
		mv := float64(sh) * finalCloses[sym]
		var costBasis float64
		if a, ok := acc.pos[sym]; ok {
			for _, l := range a.lots {
				costBasis += l.cost
			}
		}
		unrealized := mv - costBasis
		contrib := (realized + unrealized) / initialCash
		finalWeight := 0.0
		if equity > 0 {
			finalWeight = mv / equity
		}
		out = append(out, Attribution{Symbol: sym, Contribution: contrib, FinalWeight: finalWeight})
	}
	return out
}

// correlationMatrix computes pairwise Pearson correlation of symbols' return
// series over the aligned timeline (Requirement 6.6, P2). Symbols with fewer
// than 2 valid returns or zero variance get correlation 0 off-diagonal, 1 on
// the diagonal.
func correlationMatrix(aligned AlignedData, symbols []string) [][]float64 {
	rets := make(map[string][]float64, len(symbols))
	for _, sym := range symbols {
		pts := aligned.Points[sym]
		var r []float64
		for i := 1; i < len(pts); i++ {
			if pts[i-1].Close > 0 {
				r = append(r, pts[i].Close/pts[i-1].Close-1)
			}
		}
		rets[sym] = r
	}
	n := len(symbols)
	m := make([][]float64, n)
	for i := range m {
		m[i] = make([]float64, n)
		m[i][i] = 1
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			c := pearson(rets[symbols[i]], rets[symbols[j]])
			m[i][j] = c
			m[j][i] = c
		}
	}
	return m
}

// pearson returns the Pearson correlation of two equal-length-ish series,
// using the common prefix length. Returns 0 when undefined.
func pearson(a, b []float64) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	if n < 2 {
		return 0
	}
	a, b = a[:n], b[:n]
	var ma, mb float64
	for i := 0; i < n; i++ {
		ma += a[i]
		mb += b[i]
	}
	ma /= float64(n)
	mb /= float64(n)
	var cov, va, vb float64
	for i := 0; i < n; i++ {
		da, db := a[i]-ma, b[i]-mb
		cov += da * db
		va += da * da
		vb += db * db
	}
	if va <= 0 || vb <= 0 {
		return 0
	}
	c := cov / math.Sqrt(va*vb)
	if c > 1 {
		c = 1
	} else if c < -1 {
		c = -1
	}
	return c
}
