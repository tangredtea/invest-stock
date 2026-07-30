package backtest

// portfolioAccount holds a unified cash pool plus a per-symbol *account that
// reuses the existing FIFO position / T+1 sellable semantics (Requirement 3.1).
// The per-symbol account.cash field is NOT used as a source of truth; the
// portfolio-level cash scalar is authoritative.
type portfolioAccount struct {
	cash   float64
	pos    map[string]*account
	syms   []string // stable, sorted order (determinism)
	trades []PortfolioTrade
}

func newPortfolioAccount(initialCash float64, symbols []string) *portfolioAccount {
	pos := make(map[string]*account, len(symbols))
	for _, s := range symbols {
		pos[s] = newAccount(0) // per-symbol account tracks positions only
	}
	return &portfolioAccount{cash: initialCash, pos: pos, syms: symbols}
}

// shares returns the total held quantity for a symbol.
func (p *portfolioAccount) shares(sym string) int {
	if a, ok := p.pos[sym]; ok {
		return a.shares()
	}
	return 0
}

// sellable returns the quantity of a symbol sellable on bar curIdx under T+1.
func (p *portfolioAccount) sellable(sym string, curIdx int, tPlus1 bool) int {
	if a, ok := p.pos[sym]; ok {
		return a.sellable(curIdx, tPlus1)
	}
	return 0
}

// buy records a buy: reuse the per-symbol account's FIFO lot, deduct unified cash.
// cost is the buy-side cost (commission). Caller must ensure cash suffices.
func (p *portfolioAccount) buy(sym string, qty, buyIdx int, fillPrice, cost float64) {
	a := p.pos[sym]
	a.addLot(qty, buyIdx, fillPrice, cost)
	p.cash -= float64(qty)*fillPrice + cost
}

// sell records a sell: reuse the per-symbol account's FIFO sell, credit unified
// cash with proceeds (turnover minus sell-side costs). Returns the FIFO cost
// basis of the sold shares.
func (p *portfolioAccount) sell(sym string, qty, curIdx int, fillPrice, sellCost float64, tPlus1 bool) (costBasis float64) {
	a := p.pos[sym]
	costBasis = a.sellFIFO(qty, curIdx, tPlus1)
	p.cash += float64(qty)*fillPrice - sellCost
	return costBasis
}

// marketValue returns the total holding value across symbols at the given prices.
func (p *portfolioAccount) marketValue(prices map[string]float64) float64 {
	var mv float64
	for _, sym := range p.syms {
		mv += float64(p.shares(sym)) * prices[sym]
	}
	return mv
}

// equity returns cash + total market value (Requirement 3.6).
func (p *portfolioAccount) equity(prices map[string]float64) float64 {
	return p.cash + p.marketValue(prices)
}

// weight returns a symbol's market-value weight in the portfolio.
func (p *portfolioAccount) weight(sym string, prices map[string]float64) float64 {
	return p.weightWithEquity(sym, prices, p.equity(prices))
}

func (p *portfolioAccount) weightWithEquity(sym string, prices map[string]float64, equity float64) float64 {
	if equity <= 0 {
		return 0
	}
	return float64(p.shares(sym)) * prices[sym] / equity
}

// costAvg returns the average cost per share for a symbol's current holding,
// or 0 if no position.
func (p *portfolioAccount) costAvg(sym string) float64 {
	a, ok := p.pos[sym]
	if !ok {
		return 0
	}
	sh := a.shares()
	if sh == 0 {
		return 0
	}
	var totalCost float64
	for _, l := range a.lots {
		totalCost += l.cost
	}
	return totalCost / float64(sh)
}
