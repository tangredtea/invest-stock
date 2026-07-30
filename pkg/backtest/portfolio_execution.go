package backtest

import (
	"math"
	"sort"
)

// priceMode selects how the base (reference) price for a fill is determined.
type priceMode int

const (
	priceClose priceMode = iota // current bar's close
	priceOpen                   // current bar's open (for executing pending next-open orders)
)

// matchOrders matches orders (sells first, then buys), respecting cash and T+1,
// and returns the total cost incurred (Requirements 3.3, 3.5).
func (e PortfolioEngine) matchOrders(orders []order, acc *portfolioAccount, aligned AlignedData, prices map[string]float64, cfg PortfolioConfig, t int, unfilled *[]PortfolioUnfilled, mode priceMode) float64 {
	// Stable order: sells first, then buys, each by symbol order.
	sells := make([]order, 0, len(orders))
	buys := make([]order, 0, len(orders))
	for _, o := range orders {
		if o.action == Sell {
			sells = append(sells, o)
		} else {
			buys = append(buys, o)
		}
	}
	sort.SliceStable(sells, func(i, j int) bool { return sells[i].sym < sells[j].sym })
	sort.SliceStable(buys, func(i, j int) bool { return buys[i].sym < buys[j].sym })

	var totalCost float64
	for _, o := range sells {
		totalCost += e.execSell(o, acc, aligned, prices, cfg, t, unfilled, mode)
	}
	for _, o := range buys {
		totalCost += e.execBuy(o, acc, aligned, prices, cfg, t, unfilled, mode)
	}
	return totalCost
}

// basePriceFor returns the base price for a fill given the price mode.
func (e PortfolioEngine) basePriceFor(aligned AlignedData, sym string, mode priceMode, t int) (float64, bool) {
	pt := aligned.Points[sym][t]
	if !pt.Valid {
		return 0, false
	}
	if mode == priceOpen {
		return pt.Open, true
	}
	return pt.Close, true
}

func (e PortfolioEngine) execBuy(o order, acc *portfolioAccount, aligned AlignedData, prices map[string]float64, cfg PortfolioConfig, t int, unfilled *[]PortfolioUnfilled, mode priceMode) float64 {
	base, ok := e.basePriceFor(aligned, o.sym, mode, t)
	if !ok || base <= 0 {
		*unfilled = append(*unfilled, PortfolioUnfilled{Symbol: o.sym, UnfilledOrder: UnfilledOrder{Date: aligned.Timeline[t], Action: "buy", Reason: "no_next_bar", Qty: o.qty}})
		return 0
	}
	fillPrice := cfg.Base.Cost.FillPrice(base, Buy)
	if fillPrice <= 0 {
		return 0
	}
	// Largest affordable integer-lot quantity (Requirement 3.5).
	qty := o.qty
	maxAfford := int(acc.cash/(fillPrice*(1+cfg.Base.Cost.CommissionRate))/100) * 100
	if qty > maxAfford {
		rejected := qty - maxAfford
		if rejected > 0 {
			*unfilled = append(*unfilled, PortfolioUnfilled{Symbol: o.sym, UnfilledOrder: UnfilledOrder{Date: aligned.Timeline[t], Action: "buy", Reason: "insufficient_cash", Qty: rejected}})
		}
		qty = maxAfford
	}
	if qty < 100 {
		return 0
	}
	turnover := float64(qty) * fillPrice
	commission := cfg.Base.Cost.Commission(turnover)
	if turnover+commission > acc.cash {
		return 0
	}
	slippage := math.Abs(fillPrice-base) * float64(qty)
	acc.buy(o.sym, qty, t, fillPrice, commission)
	tr := TradeRecord{Date: aligned.Timeline[t], Action: "buy", Price: fillPrice, BasePrice: base, Qty: qty, Turnover: turnover, Commission: commission, Slippage: slippage, CostTotal: commission + slippage}
	acc.trades = append(acc.trades, PortfolioTrade{Symbol: o.sym, TradeRecord: tr, Rebalance: o.rebalance})
	return commission + slippage
}

func (e PortfolioEngine) execSell(o order, acc *portfolioAccount, aligned AlignedData, prices map[string]float64, cfg PortfolioConfig, t int, unfilled *[]PortfolioUnfilled, mode priceMode) float64 {
	base, ok := e.basePriceFor(aligned, o.sym, mode, t)
	if !ok || base <= 0 {
		*unfilled = append(*unfilled, PortfolioUnfilled{Symbol: o.sym, UnfilledOrder: UnfilledOrder{Date: aligned.Timeline[t], Action: "sell", Reason: "no_next_bar", Qty: o.qty}})
		return 0
	}
	fillPrice := cfg.Base.Cost.FillPrice(base, Sell)
	if fillPrice <= 0 {
		return 0
	}
	qty := o.qty
	sellable := acc.sellable(o.sym, t, cfg.Base.TPlus1)
	if qty > sellable {
		rejected := qty - sellable
		*unfilled = append(*unfilled, PortfolioUnfilled{Symbol: o.sym, UnfilledOrder: UnfilledOrder{Date: aligned.Timeline[t], Action: "sell", Reason: "tplus1_rejected", Qty: rejected}})
		qty = sellable
	}
	if qty <= 0 {
		return 0
	}
	turnover := float64(qty) * fillPrice
	commission := cfg.Base.Cost.Commission(turnover)
	stampTax := cfg.Base.Cost.StampTax(turnover, Sell)
	slippage := math.Abs(fillPrice-base) * float64(qty)
	costBasis := acc.sell(o.sym, qty, t, fillPrice, commission+stampTax, cfg.Base.TPlus1)
	realized := (turnover - commission - stampTax) - costBasis
	tr := TradeRecord{Date: aligned.Timeline[t], Action: "sell", Price: fillPrice, BasePrice: base, Qty: qty, Turnover: turnover, Commission: commission, StampTax: stampTax, Slippage: slippage, CostTotal: commission + stampTax + slippage, RealizedPL: realized}
	acc.trades = append(acc.trades, PortfolioTrade{Symbol: o.sym, TradeRecord: tr, Rebalance: o.rebalance})
	return commission + stampTax + slippage
}
