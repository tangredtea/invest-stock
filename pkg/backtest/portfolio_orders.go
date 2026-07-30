package backtest

// weightOrders translates target weights into integer-lot buy/sell orders
// (Requirements 3.4, 4.3).
func (e PortfolioEngine) weightOrders(targetW map[string]float64, acc *portfolioAccount, prices map[string]float64, cfg PortfolioConfig, t int) []order {
	equity := acc.equity(prices)
	var orders []order
	for _, sym := range acc.syms {
		w, ok := targetW[sym]
		if !ok {
			continue
		}
		price := prices[sym]
		if price <= 0 {
			continue // suspended/missing: skip (Requirements 1.5, 4.8)
		}
		targetMV := w * equity
		targetShares := int(targetMV/price/100) * 100
		delta := targetShares - acc.shares(sym)
		if delta >= 100 {
			orders = append(orders, order{sym: sym, action: Buy, qty: delta, rebalance: true})
		} else if delta <= -100 {
			orders = append(orders, order{sym: sym, action: Sell, qty: -delta, rebalance: true})
		}
	}
	return orders
}

// signalOrders translates per-symbol signals into orders, sizing buys by a
// per-symbol share of available cash.
func (e PortfolioEngine) signalOrders(dec PortfolioDecision, acc *portfolioAccount, aligned AlignedData, prices map[string]float64, cfg PortfolioConfig, t int) []order {
	var orders []order
	for _, sym := range acc.syms {
		d, ok := dec.Signals[sym]
		if !ok || d.Action == Hold {
			continue
		}
		price := prices[sym]
		if price <= 0 {
			continue
		}
		switch d.Action {
		case Buy:
			// Budget: equal share of equity across symbols.
			budget := acc.equity(prices) / float64(len(acc.syms))
			qty := int(budget/price/100) * 100
			if qty >= 100 {
				orders = append(orders, order{sym: sym, action: Buy, qty: qty})
			}
		case Sell:
			sh := acc.shares(sym)
			if sh > 0 {
				orders = append(orders, order{sym: sym, action: Sell, qty: sh})
			}
		}
	}
	return orders
}

// updatePeakClose tracks the highest close since entry per held symbol.
func (e PortfolioEngine) updatePeakClose(peakClose map[string]float64, acc *portfolioAccount, prices map[string]float64) {
	for _, sym := range acc.syms {
		if acc.shares(sym) > 0 {
			if prices[sym] > peakClose[sym] {
				peakClose[sym] = prices[sym]
			}
		} else {
			delete(peakClose, sym)
		}
	}
}

// riskOrders generates forced stop-loss / take-profit / max-drawdown orders
// (Requirements 5.1, 5.2, 5.3).
func (e PortfolioEngine) riskOrders(cfg PortfolioConfig, acc *portfolioAccount, aligned AlignedData, prices, peakClose map[string]float64, peak float64, t int) []order {
	var orders []order
	r := cfg.Risk
	for _, sym := range acc.syms {
		sh := acc.shares(sym)
		if sh == 0 || !aligned.Points[sym][t].Valid {
			continue
		}
		price := prices[sym]
		// Stop-loss (Requirement 5.1).
		if r.StopLossEnabled {
			var ref float64
			if r.StopLossRef == StopLossByPeak {
				ref = peakClose[sym]
			} else {
				ref = acc.costAvg(sym)
			}
			if ref > 0 && (ref-price)/ref >= r.StopLossPct {
				orders = append(orders, order{sym: sym, action: Sell, qty: sh})
				continue
			}
		}
		// Take-profit (Requirement 5.2).
		if r.TakeProfitEnabled {
			cost := acc.costAvg(sym)
			if cost > 0 && (price-cost)/cost >= r.TakeProfitPct {
				orders = append(orders, order{sym: sym, action: Sell, qty: sh})
			}
		}
	}
	// Max-drawdown guard (Requirement 5.3).
	if r.MaxDDGuardEnabled && peak > 0 {
		eq := acc.equity(prices)
		if (peak-eq)/peak > r.MaxDDThreshold {
			mv := acc.marketValue(prices)
			targetMV := r.MaxDDTargetExposure * eq
			if mv > targetMV {
				// Reduce each holding proportionally.
				scale := 0.0
				if mv > 0 {
					scale = targetMV / mv
				}
				for _, sym := range acc.syms {
					sh := acc.shares(sym)
					if sh == 0 || !aligned.Points[sym][t].Valid {
						continue
					}
					keep := int(float64(sh)*scale/100) * 100
					sell := sh - keep
					if sell > 0 {
						orders = append(orders, order{sym: sym, action: Sell, qty: sell})
					}
				}
			}
		}
	}
	return orders
}
