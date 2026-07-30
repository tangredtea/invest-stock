package main

import (
	"invest/pkg/backtest"
	"invest/pkg/data"
)

// loadKlinesForBacktest resolves the code and fetches klines, returning an
// HTTP status + message on failure (Requirements 14.7).
func loadKlinesForBacktest(code string) ([]data.KLine, int, string) {
	loaded, clientErr, err := loadKLinesForCode(code)
	switch clientErr {
	case klineClientErrorInvalidCode:
		return nil, 400, "无效股票代码"
	case klineClientErrorFetchFailed:
		return nil, 400, "无法获取标的历史数据: " + err.Error()
	}
	return loaded.KLines, 0, ""
}

// runOneBacktest constructs the strategy and runs the engine, mapping errors to
// HTTP status codes (Requirements 14.5, 14.6).
func runOneBacktest(req backtestReq, klines []data.KLine) (backtest.Result, int, string) {
	cfg, err := req.Config.toConfig()
	if err != nil {
		return backtest.Result{}, 400, err.Error()
	}
	s, err := backtest.Default.New(req.Strategy, req.Params)
	if err != nil {
		return backtest.Result{}, 400, err.Error()
	}
	res, err := backtest.Engine{}.Run(klines, s, cfg)
	if err != nil {
		return backtest.Result{}, 400, err.Error()
	}
	res.Params = req.Params
	return res, 0, ""
}
