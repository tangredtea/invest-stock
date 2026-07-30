package backtest

import (
	"fmt"

	"invest/pkg/data"
	"invest/pkg/indicator"
)

// Context is the read-only context the engine provides to a strategy on bar Index.
// Strategies may only read market data at indices <= Index; reading the future
// panics (Requirement 1.9), which surfaces look-ahead bugs in tests.
//
// Cash and Shares expose a read-only snapshot of the account state *before* the
// current bar's order is matched, so position-aware strategies (full-position
// entry/exit, grid sizing) can size their orders.
type Context struct {
	Index  int
	Klines []data.KLine
	Series indicator.Series
	Cash   float64 // available cash before this bar's fill
	Shares int     // total holdings before this bar's fill
}

// Bar returns the i-th K线; panics if i > Index (look-ahead violation).
func (c *Context) Bar(i int) data.KLine {
	c.checkHistoricalIndex("Bar", i, len(c.Klines))
	return c.Klines[i]
}

// Ind returns indicator field value at index i; panics if i > Index.
func (c *Context) Ind(field IndField, i int) float64 {
	if i < 0 || i > c.Index {
		panic(fmt.Sprintf("backtest: look-ahead access Ind(%d,%d) at index %d", field, i, c.Index))
	}
	s := c.Series
	fieldLen := indicatorFieldLen(s, field)
	if fieldLen < 0 {
		return 0
	}
	c.checkHistoricalIndex(fmt.Sprintf("Ind(%d)", field), i, fieldLen)
	return indicatorFieldAt(s, field, i)
}

func (c *Context) checkHistoricalIndex(op string, i, n int) {
	if i < 0 || i > c.Index {
		panic(fmt.Sprintf("backtest: look-ahead access %s(%d) at index %d", op, i, c.Index))
	}
	if i >= n {
		panic(fmt.Sprintf("backtest: %s(%d) out of range for %d bars at index %d", op, i, n, c.Index))
	}
}
