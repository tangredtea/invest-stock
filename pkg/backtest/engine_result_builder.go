package backtest

import "invest/pkg/data"

func buildEngineResult(s Strategy, cfg Config, window []data.KLine, acc *account, curves engineCurves, unfilled []UnfilledOrder) Result {
	n := len(window)
	calendarDays := int(window[n-1].Date.Sub(window[0].Date).Hours() / 24)
	metrics, _ := Calculate(curves.equity, curves.dates, acc.trades, cfg.InitialCash, calendarDays, cfg.RiskFreeRate)

	return Result{
		StrategyName: s.Name(),
		Equity:       curves.equity,
		Drawdown:     curves.drawdown,
		NetValue:     curves.netValue,
		Dates:        curves.dates,
		Trades:       acc.trades,
		Unfilled:     unfilled,
		Metrics:      metrics,
	}
}
