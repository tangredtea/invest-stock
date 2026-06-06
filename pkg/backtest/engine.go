package backtest

import (
	"fmt"
	"time"

	"invest/pkg/data"
	"invest/pkg/indicator"
)

// platformMinBars is the platform hard lower bound on participating bars,
// matching the existing RunBacktest threshold (Requirement 19.8).
const platformMinBars = 60

// Engine is a stateless, deterministic backtest executor (Requirements 5.7, 5.8).
type Engine struct{}

// Run executes one backtest (Requirements 5.1, 5.2, 5.6, 5.9, 5.10). Any
// validation failure returns an error and produces no Result.
func (e Engine) Run(klines []data.KLine, s Strategy, cfg Config) (Result, error) {
	// 1. Strategy and config validation (Requirement 5.9).
	if s == nil {
		return Result{}, fmt.Errorf("策略不能为空")
	}
	if err := cfg.Validate(); err != nil {
		return Result{}, err
	}

	// 2. K线 time-series validation: strictly ascending, no duplicates (Requirement 5.10).
	for i := 1; i < len(klines); i++ {
		if !klines[i].Date.After(klines[i-1].Date) {
			return Result{}, fmt.Errorf("K线时间序列无效: 第 %d 根日期非严格升序或重复", i)
		}
	}

	// 3. Date-range filtering (Requirements 6.3–6.6).
	window := filterByDate(klines, cfg.StartDate, cfg.EndDate)
	if (cfg.StartDate != nil || cfg.EndDate != nil) && len(window) == 0 {
		return Result{}, fmt.Errorf("指定日期区间内无可用 K线数据")
	}

	// 4. Minimum-bar validation (Requirements 5.5, 19.8).
	need := s.MinBars()
	if need < platformMinBars {
		need = platformMinBars
	}
	if len(window) < need {
		return Result{}, fmt.Errorf("K线数量不足: 实际 %d 根, 至少需要 %d 根", len(window), need)
	}

	// 5. Indicator precompute, aligned with the window.
	closes := make([]float64, len(window))
	for i, k := range window {
		closes[i] = k.Close
	}
	series := indicator.ComputeSeries(closes)

	// 6. Event loop.
	return e.runLoop(window, series, s, cfg), nil
}

// filterByDate returns the sub-slice whose Date falls in the optional [start,end]
// closed interval (Requirements 6.3, 6.4).
func filterByDate(klines []data.KLine, start, end *time.Time) []data.KLine {
	if start == nil && end == nil {
		return klines
	}
	out := make([]data.KLine, 0, len(klines))
	for _, k := range klines {
		if start != nil && k.Date.Before(*start) {
			continue
		}
		if end != nil && k.Date.After(*end) {
			continue
		}
		out = append(out, k)
	}
	return out
}

// runLoop drives the strategy bar by bar, matching fills per the Fill rule,
// applying costs and T+1, recording trades and curves, then computing metrics.
func (e Engine) runLoop(window []data.KLine, series indicator.Series, s Strategy, cfg Config) Result {
	n := len(window)
	acc := newAccount(cfg.InitialCash)

	equity := make([]float64, n)
	drawdown := make([]float64, n)
	netValue := make([]float64, n)
	dates := make([]time.Time, n)
	var unfilled []UnfilledOrder

	peak := cfg.InitialCash

	for i := 0; i < n; i++ {
		ctx := &Context{
			Index: i, Klines: window, Series: series,
			Cash: acc.cash, Shares: acc.shares(),
		}
		dec := s.Decide(ctx)

		if dec.Action != Hold {
			e.execute(acc, &unfilled, window, i, dec, cfg)
		}

		// Mark-to-market at the current bar's close (Requirement 19.6).
		price := window[i].Close
		eq := acc.equity(price)
		equity[i] = eq
		dates[i] = window[i].Date
		netValue[i] = eq / cfg.InitialCash
		if eq > peak {
			peak = eq
		}
		if peak > 0 {
			drawdown[i] = (peak - eq) / peak
			if drawdown[i] < 0 {
				drawdown[i] = 0
			}
		}
	}

	calendarDays := int(window[n-1].Date.Sub(window[0].Date).Hours() / 24)
	metrics, _ := Calculate(equity, dates, acc.trades, cfg.InitialCash, calendarDays, cfg.RiskFreeRate)

	return Result{
		StrategyName: s.Name(),
		Equity:       equity,
		Drawdown:     drawdown,
		NetValue:     netValue,
		Dates:        dates,
		Trades:       acc.trades,
		Unfilled:     unfilled,
		Metrics:      metrics,
	}
}

// execute resolves a buy/sell decision into a fill (or an unfilled record),
// applying Fill rule, slippage, costs, cash and T+1 constraints
// (Requirements 5.3, 5.4, 6.7, 6.8, 7.6–7.10, 8.1, 8.2).
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
