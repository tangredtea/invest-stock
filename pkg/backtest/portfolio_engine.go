package backtest

import (
	"fmt"
	"math"
	"sort"

	"invest/pkg/indicator"
)

// PortfolioEngine is a stateless, deterministic portfolio backtest executor
// (Requirements 3.9, 3.10).
type PortfolioEngine struct{}

// Run executes one portfolio backtest (Requirement 3). Any validation failure
// returns an error and produces no PortfolioResult.
func (e PortfolioEngine) Run(aligned AlignedData, s PortfolioStrategy, cfg PortfolioConfig) (PortfolioResult, error) {
	if s == nil {
		return PortfolioResult{}, fmt.Errorf("组合策略不能为空")
	}
	if err := cfg.Validate(); err != nil {
		return PortfolioResult{}, err
	}
	n := len(aligned.Timeline)
	if n == 0 {
		return PortfolioResult{}, fmt.Errorf("统一时间轴为空")
	}
	if n < platformMinBars {
		return PortfolioResult{}, fmt.Errorf("时间轴长度不足: 实际 %d, 至少需要 %d", n, platformMinBars)
	}

	symbols := aligned.Symbols

	// Precompute per-symbol indicator series on aligned closes (Requirement 3.3).
	series := make(map[string]indicator.Series, len(symbols))
	for _, sym := range symbols {
		closes := make([]float64, n)
		for i, p := range aligned.Points[sym] {
			closes[i] = p.Close
		}
		series[sym] = indicator.ComputeSeries(closes)
	}

	acc := newPortfolioAccount(cfg.Base.InitialCash, symbols)

	equity := make([]float64, n)
	drawdown := make([]float64, n)
	netValue := make([]float64, n)
	cashWeight := make([]float64, n)
	weights := make(map[string][]float64, len(symbols))
	for _, sym := range symbols {
		weights[sym] = make([]float64, n)
	}
	peakClose := make(map[string]float64) // highest close since entry, for stop-loss
	var unfilled []PortfolioUnfilled

	peak := cfg.Base.InitialCash
	lastRebalanceIdx := 0
	rebalanceCount := 0
	var rebalanceCost float64
	var pending []order // orders decided at t-1, executed at t's open (FillNextOpen)

	for t := 0; t < n; t++ {
		validCloses := e.validCloses(aligned, t)

		// Execute pending orders (decided last bar) at THIS bar's open under
		// the next-open fill rule, before any new decision. This keeps
		// equity[0] == initial cash (Requirement 3.8).
		if cfg.Base.FillRule == FillNextOpen && len(pending) > 0 {
			rebalanceCost += e.matchOrders(pending, acc, aligned, validCloses, cfg, t, &unfilled, priceOpen)
			pending = nil
		}

		// Build read-only context snapshot.
		shares := make(map[string]int, len(symbols))
		for _, sym := range symbols {
			shares[sym] = acc.shares(sym)
		}
		ctx := &PortfolioContext{
			Index: t, Timeline: aligned.Timeline, Symbols: symbols,
			aligned: &aligned, series: series,
			Cash: acc.cash, Shares: shares, Equity: acc.equity(validCloses),
		}

		dec := s.Decide(ctx)

		// Resolve this bar's target weights (Requirement 5.6, 5.7).
		targetW, err := e.resolveTargetWeights(dec, ctx, aligned, series, cfg, t)
		if err != nil {
			return PortfolioResult{}, err
		}

		// Decide whether to act this bar.
		doRebalance, isRebalance := e.shouldRebalance(cfg, acc, validCloses, targetW, t, lastRebalanceIdx)

		var orders []order
		if dec.Kind == DecisionSignals {
			orders = e.signalOrders(dec, acc, aligned, validCloses, cfg, t)
		} else if t == 0 || doRebalance {
			orders = e.weightOrders(targetW, acc, validCloses, cfg, t)
			if doRebalance {
				isRebalance = true
			}
		}

		// Risk hooks: stop-loss / take-profit / max-drawdown forced orders.
		e.updatePeakClose(peakClose, acc, validCloses)
		riskOrders := e.riskOrders(cfg, acc, aligned, validCloses, peakClose, peak, t)
		orders = append(orders, riskOrders...)

		if isRebalance && len(orders) > 0 {
			rebalanceCount++
			lastRebalanceIdx = t
		}

		// Execute per fill rule: close => now at this bar's close;
		// next-open => queue as pending for next bar's open (Requirement 5.3).
		if cfg.Base.FillRule == FillClose {
			rebalanceCost += e.matchOrders(orders, acc, aligned, validCloses, cfg, t, &unfilled, priceClose)
		} else {
			pending = orders
		}

		// Mark-to-market at this bar's valid closes (Requirement 3.6).
		eq := acc.equity(validCloses)
		equity[t] = eq
		netValue[t] = eq / cfg.Base.InitialCash
		if eq > peak {
			peak = eq
		}
		if peak > 0 {
			dd := (peak - eq) / peak
			if dd < 0 {
				dd = 0
			}
			drawdown[t] = dd
		}
		for _, sym := range symbols {
			weights[sym][t] = acc.weight(sym, validCloses)
		}
		if eq > 0 {
			cashWeight[t] = acc.cash / eq
		}
	}

	// Any pending orders at the end (next-open with no next bar) are dropped as
	// unfilled (Requirement 5.4).
	for _, o := range pending {
		*(&unfilled) = append(unfilled, PortfolioUnfilled{Symbol: o.sym, UnfilledOrder: UnfilledOrder{Date: aligned.Timeline[n-1], Action: o.action.String(), Reason: "no_next_bar", Qty: o.qty}})
	}

	// Performance metrics on the portfolio equity curve (Requirement 6.1).
	calendarDays := int(aligned.Timeline[n-1].Sub(aligned.Timeline[0]).Hours() / 24)
	flatTrades := make([]TradeRecord, len(acc.trades))
	for i, pt := range acc.trades {
		flatTrades[i] = pt.TradeRecord
	}
	metrics, _ := Calculate(equity, aligned.Timeline, flatTrades, cfg.Base.InitialCash, calendarDays, cfg.Base.RiskFreeRate)

	finalCloses := e.validCloses(aligned, n-1)
	attribution := computeAttribution(acc, symbols, finalCloses, cfg.Base.InitialCash)

	res := PortfolioResult{
		Symbols: symbols, Dates: aligned.Timeline,
		Equity: equity, Drawdown: drawdown, NetValue: netValue,
		Weights: weights, CashWeight: cashWeight,
		Trades: acc.trades, Unfilled: unfilled, Metrics: metrics,
		Attribution: attribution, RebalanceCount: rebalanceCount, RebalanceCost: rebalanceCost,
	}
	if cfg.Correlation {
		res.Correlation = correlationMatrix(aligned, symbols)
	}
	return res, nil
}

// order is an internal buy/sell instruction for one symbol.
type order struct {
	sym       string
	action    Action
	qty       int
	rebalance bool
}

// validCloses returns each symbol's effective close at index t (last valid close
// for suspended symbols; 0 for missing) (Requirement 1.5).
func (e PortfolioEngine) validCloses(aligned AlignedData, t int) map[string]float64 {
	out := make(map[string]float64, len(aligned.Symbols))
	for _, sym := range aligned.Symbols {
		out[sym] = aligned.Points[sym][t].Close
	}
	return out
}

// resolveTargetWeights computes the target weights for bar t from the decision
// and the configured weight scheme, then applies position caps and cash reserve
// (Requirements 5.4, 5.5, 5.6, 5.7).
func (e PortfolioEngine) resolveTargetWeights(dec PortfolioDecision, ctx *PortfolioContext, aligned AlignedData, series map[string]indicator.Series, cfg PortfolioConfig, t int) (map[string]float64, error) {
	tradable := make([]string, 0, len(aligned.Symbols))
	for _, sym := range aligned.Symbols {
		if aligned.Points[sym][t].Valid {
			tradable = append(tradable, sym)
		}
	}

	w := make(map[string]float64)
	if dec.Kind == DecisionWeights && len(dec.Weights) > 0 {
		// Validate each weight in [0,1] and sum <= 1 (Requirements 2.9, 2.5).
		var sum float64
		for sym, weight := range dec.Weights {
			if weight < 0 || weight > 1 {
				return nil, fmt.Errorf("标的 %s 目标权重 %g 不在 [0,1]", sym, weight)
			}
			sum += weight
		}
		if sum > 1+weightSumTolerance {
			return nil, fmt.Errorf("目标权重之和 %g 超过 1", sum)
		}
		for sym, weight := range dec.Weights {
			w[sym] = weight
		}
	} else {
		// Signal mode, or a target-weight strategy deferring to the configured
		// scheme (nil weights): weights come from the scheme (Requirement 5.6).
		w = e.schemeWeights(cfg, tradable, aligned, series, t)
	}

	// Apply per-symbol cap (Requirement 5.4).
	for sym := range w {
		if cap, ok := cfg.Risk.PerSymbolCap[sym]; ok && w[sym] > cap {
			w[sym] = cap
		}
	}
	// Apply cash reserve: scale down if total exceeds 1 - reserve (Requirement 5.5).
	alloc := 1.0 - cfg.Risk.CashReserve
	var sum float64
	for _, weight := range w {
		sum += weight
	}
	if sum > alloc && sum > 0 {
		scale := alloc / sum
		for sym := range w {
			w[sym] *= scale
		}
	}
	return w, nil
}

// schemeWeights computes target weights per the configured WeightScheme over
// the tradable symbols (Requirements 5.6, 5.7, 5.9).
func (e PortfolioEngine) schemeWeights(cfg PortfolioConfig, tradable []string, aligned AlignedData, series map[string]indicator.Series, t int) map[string]float64 {
	w := make(map[string]float64, len(tradable))
	alloc := 1.0 - cfg.Risk.CashReserve
	switch cfg.Scheme {
	case WeightSpecified:
		for _, sym := range tradable {
			w[sym] = cfg.Weights[sym]
		}
	case WeightInverseVol:
		invVol := make(map[string]float64, len(tradable))
		var total float64
		for _, sym := range tradable {
			v := e.recentVol(aligned, sym, t, cfg.InverseVolWindow)
			if v > 0 {
				invVol[sym] = 1 / v
			}
			total += invVol[sym]
		}
		if total > 0 {
			for _, sym := range tradable {
				w[sym] = alloc * invVol[sym] / total
			}
		} else {
			w = equalWeights(tradable, alloc)
		}
	default: // WeightEqual
		w = equalWeights(tradable, alloc)
	}
	return w
}

// equalWeights assigns an equal share of alloc to each symbol (Requirement 5.7).
func equalWeights(syms []string, alloc float64) map[string]float64 {
	w := make(map[string]float64, len(syms))
	if len(syms) == 0 {
		return w
	}
	each := alloc / float64(len(syms))
	for _, sym := range syms {
		w[sym] = each
	}
	return w
}

// recentVol returns the sample std of recent daily returns for a symbol.
func (e PortfolioEngine) recentVol(aligned AlignedData, sym string, t, window int) float64 {
	pts := aligned.Points[sym]
	start := t - window + 1
	if start < 1 {
		start = 1
	}
	var rets []float64
	for i := start; i <= t; i++ {
		if pts[i-1].Close > 0 {
			rets = append(rets, pts[i].Close/pts[i-1].Close-1)
		}
	}
	_, std := meanSampleStd(rets)
	return std
}

// shouldRebalance reports whether to rebalance at bar t (periodic or threshold),
// collapsing simultaneous triggers into one (Requirements 4.1, 4.2, 4.5).
func (e PortfolioEngine) shouldRebalance(cfg PortfolioConfig, acc *portfolioAccount, prices, targetW map[string]float64, t, lastIdx int) (do, isReb bool) {
	if cfg.Rebalance.Periodic && t > 0 && (t-lastIdx) >= cfg.Rebalance.PeriodBars {
		return true, true
	}
	if cfg.Rebalance.Threshold {
		for sym, tw := range targetW {
			cur := acc.weight(sym, prices)
			if math.Abs(cur-tw) > cfg.Rebalance.ThresholdValue {
				return true, true
			}
		}
	}
	return false, false
}

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
