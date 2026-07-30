package backtest

func (e PortfolioEngine) buildPortfolioResult(
	aligned AlignedData,
	symbols []string,
	cfg PortfolioConfig,
	acc *portfolioAccount,
	curves portfolioCurves,
	unfilled []PortfolioUnfilled,
	rebalanceCount int,
	rebalanceCost float64,
) PortfolioResult {
	n := len(aligned.Timeline)
	calendarDays := int(aligned.Timeline[n-1].Sub(aligned.Timeline[0]).Hours() / 24)
	metrics, _ := Calculate(
		curves.equity,
		aligned.Timeline,
		flattenPortfolioTrades(acc.trades),
		cfg.Base.InitialCash,
		calendarDays,
		cfg.Base.RiskFreeRate,
	)

	finalCloses := e.validCloses(aligned, n-1)
	attribution := computeAttribution(acc, symbols, finalCloses, cfg.Base.InitialCash)

	return PortfolioResult{
		Symbols:        symbols,
		Dates:          aligned.Timeline,
		Equity:         curves.equity,
		Drawdown:       curves.drawdown,
		NetValue:       curves.netValue,
		Weights:        curves.weights,
		CashWeight:     curves.cashWeight,
		Trades:         acc.trades,
		Unfilled:       unfilled,
		Metrics:        metrics,
		Attribution:    attribution,
		RebalanceCount: rebalanceCount,
		RebalanceCost:  rebalanceCost,
	}
}

func flattenPortfolioTrades(trades []PortfolioTrade) []TradeRecord {
	out := make([]TradeRecord, len(trades))
	for i, trade := range trades {
		out[i] = trade.TradeRecord
	}
	return out
}
