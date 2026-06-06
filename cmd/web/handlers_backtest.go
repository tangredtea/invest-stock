package main

import (
	"encoding/json"
	"net/http"
	"time"

	"invest/pkg/backtest"
	_ "invest/pkg/backtest/builtin" // register builtin strategies
	"invest/pkg/data"
)

// backtestReq is one backtest request: a security code, a strategy name +
// params, and an optional config override.
type backtestReq struct {
	Code     string                          `json:"code"`
	Strategy string                          `json:"strategy"`
	Params   map[string]backtest.ParamValue  `json:"params"`
	Config   *backtestConfigReq              `json:"config"`
}

// backtestConfigReq mirrors the configurable subset of backtest.Config in JSON.
type backtestConfigReq struct {
	InitialCash    *float64 `json:"initialCash"`
	StartDate      string   `json:"startDate"` // "2006-01-02", optional
	EndDate        string   `json:"endDate"`
	FillRule       *int     `json:"fillRule"`
	TPlus1         *bool    `json:"tPlus1"`
	RiskFreeRate   *float64 `json:"riskFreeRate"`
	CommissionRate *float64 `json:"commissionRate"`
	MinCommission  *float64 `json:"minCommission"`
	StampTaxRate   *float64 `json:"stampTaxRate"`
	SlippageRatio  *float64 `json:"slippageRatio"`
}

// toConfig builds a backtest.Config from the request, starting from defaults.
func (c *backtestConfigReq) toConfig() (backtest.Config, error) {
	cfg := backtest.DefaultConfig()
	if c == nil {
		return cfg, nil
	}
	if c.InitialCash != nil {
		cfg.InitialCash = *c.InitialCash
	}
	if c.FillRule != nil {
		cfg.FillRule = backtest.FillRule(*c.FillRule)
	}
	if c.TPlus1 != nil {
		cfg.TPlus1 = *c.TPlus1
	}
	if c.RiskFreeRate != nil {
		cfg.RiskFreeRate = *c.RiskFreeRate
	}
	if c.CommissionRate != nil {
		cfg.Cost.CommissionRate = *c.CommissionRate
	}
	if c.MinCommission != nil {
		cfg.Cost.MinCommission = *c.MinCommission
	}
	if c.StampTaxRate != nil {
		cfg.Cost.StampTaxRate = *c.StampTaxRate
	}
	if c.SlippageRatio != nil {
		cfg.Cost.SlippageRatio = *c.SlippageRatio
	}
	if c.StartDate != "" {
		t, err := time.Parse("2006-01-02", c.StartDate)
		if err != nil {
			return cfg, errBadDate
		}
		cfg.StartDate = &t
	}
	if c.EndDate != "" {
		t, err := time.Parse("2006-01-02", c.EndDate)
		if err != nil {
			return cfg, errBadDate
		}
		cfg.EndDate = &t
	}
	return cfg, nil
}

var errBadDate = &apiError{"日期格式无效, 应为 2006-01-02"}

type apiError struct{ msg string }

func (e *apiError) Error() string { return e.msg }

// handleBacktest runs a single-strategy backtest (Requirement 14).
func handleBacktest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, jsonResp{Code: 405, Message: "method not allowed"})
		return
	}
	var req backtestReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "请求格式错误"})
		return
	}

	klines, status, errMsg := loadKlinesForBacktest(req.Code)
	if errMsg != "" {
		writeJSON(w, status, jsonResp{Code: status, Message: errMsg})
		return
	}

	res, status, errMsg := runOneBacktest(req, klines)
	if errMsg != "" {
		writeJSON(w, status, jsonResp{Code: status, Message: errMsg})
		return
	}
	writeJSON(w, 200, jsonResp{Code: 0, Message: "ok", Data: res})
}

// handleBacktestCompare runs multiple strategies on the same data/config and
// returns results in request order (Requirement 15).
func handleBacktestCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, jsonResp{Code: 405, Message: "method not allowed"})
		return
	}
	var req struct {
		Code      string             `json:"code"`
		Config    *backtestConfigReq `json:"config"`
		Strategies []struct {
			Strategy string                         `json:"strategy"`
			Params   map[string]backtest.ParamValue `json:"params"`
		} `json:"strategies"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "请求格式错误"})
		return
	}
	if len(req.Strategies) < 1 || len(req.Strategies) > 50 {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "策略组合数量必须在 1 到 50 之间"})
		return
	}

	klines, status, errMsg := loadKlinesForBacktest(req.Code)
	if errMsg != "" {
		writeJSON(w, status, jsonResp{Code: status, Message: errMsg})
		return
	}

	results := make([]backtest.Result, 0, len(req.Strategies))
	for i, item := range req.Strategies {
		single := backtestReq{Code: req.Code, Strategy: item.Strategy, Params: item.Params, Config: req.Config}
		res, st, em := runOneBacktest(single, klines)
		if em != "" {
			writeJSON(w, st, jsonResp{Code: st, Message: "组合 #" + itoaSafe(i) + " (" + item.Strategy + "): " + em})
			return
		}
		results = append(results, res)
	}
	writeJSON(w, 200, jsonResp{Code: 0, Message: "ok", Data: results})
}

// loadKlinesForBacktest resolves the code and fetches klines, returning an
// HTTP status + message on failure (Requirements 14.7).
func loadKlinesForBacktest(code string) ([]data.KLine, int, string) {
	secid := resolveSecID(code)
	if secid == "" {
		return nil, 400, "无效股票代码"
	}
	klines, err := data.FetchKLines(secid)
	if err != nil {
		return nil, 400, "无法获取标的历史数据: "+err.Error()
	}
	return klines, 0, ""
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

// handleListStrategies returns all registered strategies with their parameter
// definitions, so the frontend can render parameter forms (Requirement 18.2).
func handleListStrategies(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, jsonResp{Code: 0, Message: "ok", Data: backtest.Default.List()})
}

func itoaSafe(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
