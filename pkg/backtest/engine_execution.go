package backtest

import "invest/pkg/data"

// execute resolves a buy/sell decision into a fill (or an unfilled record),
// applying Fill rule, slippage, costs, cash and T+1 constraints
// (Requirements 5.3, 5.4, 6.7, 6.8, 7.6-7.10, 8.1, 8.2).
func (e Engine) execute(acc *account, unfilled *[]UnfilledOrder, window []data.KLine, i int, dec Decision, cfg Config) {
	// Determine the base (reference) price per Fill rule (Requirement 5.3).
	var base float64
	switch cfg.FillRule {
	case FillNextOpen:
		if i+1 >= len(window) {
			// No next bar: abandon the fill (Requirement 5.4).
			*unfilled = append(*unfilled, UnfilledOrder{
				Date: window[i].Date, Action: dec.Action.String(), Reason: "no_next_bar",
			})
			return
		}
		base = window[i+1].Open
	default: // FillClose
		base = window[i].Close
	}
	if base <= 0 {
		return
	}

	fillPrice := cfg.Cost.FillPrice(base, dec.Action)
	if fillPrice <= 0 {
		return
	}

	switch dec.Action {
	case Buy:
		e.executeBuy(acc, unfilled, window, i, dec, base, fillPrice, cfg)
	case Sell:
		e.executeSell(acc, unfilled, window, i, dec, base, fillPrice, cfg)
	}
}

func (e Engine) executeBuy(acc *account, unfilled *[]UnfilledOrder, window []data.KLine, i int, dec Decision, base, fillPrice float64, cfg Config) {
	// Resolve target quantity from Qty or Amount.
	qty := dec.Qty
	if qty == 0 && dec.Amount > 0 {
		qty = int(dec.Amount / fillPrice)
	}
	if qty <= 0 {
		return
	}
	turnover := float64(qty) * fillPrice
	commission := cfg.Cost.Commission(turnover)
	need := turnover + commission
	if need > acc.cash {
		// Insufficient cash: abandon, cash & holdings unchanged (Requirement 7.8).
		*unfilled = append(*unfilled, UnfilledOrder{
			Date: window[i].Date, Action: "buy", Reason: "insufficient_cash", Qty: qty,
		})
		return
	}
	slippage := absFloat(fillPrice-base) * float64(qty)
	acc.addLot(qty, i, fillPrice, commission)
	acc.trades = append(acc.trades, TradeRecord{
		Date: window[i].Date, Action: "buy", Price: fillPrice, BasePrice: base,
		Qty: qty, Turnover: turnover, Commission: commission, StampTax: 0,
		Slippage: slippage, CostTotal: commission + slippage,
	})
}

func (e Engine) executeSell(acc *account, unfilled *[]UnfilledOrder, window []data.KLine, i int, dec Decision, base, fillPrice float64, cfg Config) {
	qty := dec.Qty
	if qty == 0 && dec.Amount > 0 {
		qty = int(dec.Amount / fillPrice)
	}
	if qty <= 0 {
		return
	}
	sellable := acc.sellable(i, cfg.TPlus1)
	if qty > sellable {
		// T+1 (or holdings) restricts the sell: only fill the sellable part,
		// record the rejected remainder (Requirement 8.1).
		rejected := qty - sellable
		*unfilled = append(*unfilled, UnfilledOrder{
			Date: window[i].Date, Action: "sell", Reason: "tplus1_rejected", Qty: rejected,
		})
		qty = sellable
	}
	if qty <= 0 {
		return
	}
	turnover := float64(qty) * fillPrice
	commission := cfg.Cost.Commission(turnover)
	stampTax := cfg.Cost.StampTax(turnover, Sell)
	slippage := absFloat(fillPrice-base) * float64(qty)

	costBasis := acc.sellFIFO(qty, i, cfg.TPlus1)
	proceeds := turnover - commission - stampTax
	acc.cash += proceeds
	realized := proceeds - costBasis // net realized P/L for this close (Requirement 12.2, 12.4)

	acc.trades = append(acc.trades, TradeRecord{
		Date: window[i].Date, Action: "sell", Price: fillPrice, BasePrice: base,
		Qty: qty, Turnover: turnover, Commission: commission, StampTax: stampTax,
		Slippage: slippage, CostTotal: commission + stampTax + slippage,
		RealizedPL: realized,
	})
}

func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
