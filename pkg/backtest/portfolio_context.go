package backtest

import (
	"fmt"

	"invest/pkg/indicator"
)

func newPortfolioContext(index int, aligned AlignedData, series map[string]indicator.Series, acc *portfolioAccount, prices map[string]float64) *PortfolioContext {
	shares := make(map[string]int, len(aligned.Symbols))
	for _, sym := range aligned.Symbols {
		shares[sym] = acc.shares(sym)
	}
	return &PortfolioContext{
		Index:    index,
		Timeline: aligned.Timeline,
		Symbols:  aligned.Symbols,
		Cash:     acc.cash,
		Shares:   shares,
		Equity:   acc.equity(prices),
		aligned:  &aligned,
		series:   series,
	}
}

// CanTrade reports whether a symbol is tradable at the current point
// (not suspended, not missing) (Requirements 1.5, 2.2).
func (c *PortfolioContext) CanTrade(sym string) bool {
	pts, ok := c.aligned.Points[sym]
	if !ok || c.Index < 0 || c.Index >= len(pts) {
		return false
	}
	return pts[c.Index].Valid
}

// Bar returns a symbol's aligned price point at index i; panics if i > Index
// (look-ahead protection, Requirement 2.7).
func (c *PortfolioContext) Bar(sym string, i int) PricePoint {
	pts, ok := c.aligned.Points[sym]
	if !ok {
		panic(fmt.Sprintf("portfolio: unknown symbol %q for Bar at index %d", sym, c.Index))
	}
	c.checkHistoricalIndex(fmt.Sprintf("Bar(%s)", sym), i, len(pts))
	return pts[i]
}

// Ind returns a symbol's indicator field at index i; panics if i > Index.
func (c *PortfolioContext) Ind(sym string, field IndField, i int) float64 {
	s, ok := c.series[sym]
	if !ok {
		panic(fmt.Sprintf("portfolio: unknown symbol %q for Ind at index %d", sym, c.Index))
	}
	if i < 0 || i > c.Index {
		panic(fmt.Sprintf("portfolio: look-ahead access Ind(%s,%d)(%d) at index %d", sym, field, i, c.Index))
	}
	fieldLen := indicatorFieldLen(s, field)
	if fieldLen < 0 {
		return 0
	}
	if i >= fieldLen {
		panic(fmt.Sprintf("portfolio: Ind(%s,%d)(%d) out of range for %d bars at index %d", sym, field, i, fieldLen, c.Index))
	}
	return indicatorFieldAt(c.series[sym], field, i)
}

// ValidBars returns the number of valid (non-suspended, non-missing) price
// points for a symbol up to and including the current index (Requirement 2.11).
func (c *PortfolioContext) ValidBars(sym string) int {
	pts := c.aligned.Points[sym]
	n := 0
	for i := 0; i <= c.Index && i < len(pts); i++ {
		if pts[i].Valid {
			n++
		}
	}
	return n
}

func (c *PortfolioContext) checkHistoricalIndex(op string, i, n int) {
	if i < 0 || i > c.Index {
		panic(fmt.Sprintf("portfolio: look-ahead access %s(%d) at index %d", op, i, c.Index))
	}
	if i >= n {
		panic(fmt.Sprintf("portfolio: %s(%d) out of range for %d bars at index %d", op, i, n, c.Index))
	}
}
