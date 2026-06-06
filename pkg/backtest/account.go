package backtest

// lot is a position batch from a single buy, tagged with its buy bar index for
// T+1 sellability determination (Requirement 8.3).
type lot struct {
	qty    int
	buyIdx int
	cost   float64 // total cost basis of this lot (turnover + buy-side costs)
}

// account maintains cash, FIFO position lots, and the trade log (Requirements 6.7, 6.8).
type account struct {
	cash   float64
	lots   []lot
	trades []TradeRecord
}

func newAccount(initialCash float64) *account {
	return &account{cash: initialCash}
}

// shares returns the total held quantity.
func (a *account) shares() int {
	total := 0
	for _, l := range a.lots {
		total += l.qty
	}
	return total
}

// sellable returns the quantity that may be sold on bar curIdx (Requirements 8.3, 8.4).
// With T+1, only lots bought on an earlier bar (buyIdx < curIdx) count; without
// T+1, the full holding is sellable.
func (a *account) sellable(curIdx int, tPlus1 bool) int {
	if !tPlus1 {
		return a.shares()
	}
	total := 0
	for _, l := range a.lots {
		if l.buyIdx < curIdx {
			total += l.qty
		}
	}
	return total
}

// marketValue returns the holding value at the given price.
func (a *account) marketValue(price float64) float64 {
	return float64(a.shares()) * price
}

// equity returns total account value = cash + market value (Requirement 19.6).
func (a *account) equity(price float64) float64 {
	return a.cash + a.marketValue(price)
}

// addLot records a buy: append a position lot and deduct cash.
func (a *account) addLot(qty, buyIdx int, fillPrice, buyCost float64) {
	a.lots = append(a.lots, lot{qty: qty, buyIdx: buyIdx, cost: float64(qty)*fillPrice + buyCost})
	a.cash -= float64(qty)*fillPrice + buyCost
}

// sellFIFO removes qty shares from the oldest sellable lots (those with
// buyIdx < curIdx when tPlus1, otherwise any), returning the realized cost basis
// of the sold shares. Caller must ensure qty <= sellable(curIdx, tPlus1).
func (a *account) sellFIFO(qty, curIdx int, tPlus1 bool) (costBasis float64) {
	remaining := qty
	newLots := a.lots[:0]
	// Build a fresh slice to avoid aliasing issues while iterating.
	kept := make([]lot, 0, len(a.lots))
	for _, l := range a.lots {
		if remaining == 0 || (tPlus1 && l.buyIdx >= curIdx) {
			kept = append(kept, l)
			continue
		}
		if l.qty <= remaining {
			// Whole lot sold.
			costBasis += l.cost
			remaining -= l.qty
		} else {
			// Partial lot sold; reduce proportionally.
			perShare := l.cost / float64(l.qty)
			costBasis += perShare * float64(remaining)
			l.cost -= perShare * float64(remaining)
			l.qty -= remaining
			remaining = 0
			kept = append(kept, l)
		}
	}
	_ = newLots
	a.lots = kept
	return costBasis
}
