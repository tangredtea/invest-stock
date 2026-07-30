package main

import (
	"net/http"
)

// handlePortfolioBacktest runs a portfolio backtest (Requirement 7).
func handlePortfolioBacktest(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var req portfolioReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "请求格式错误"})
		return
	}

	res, status, errMsg := runPortfolioBacktest(&req)
	if errMsg != "" {
		writeJSON(w, status, jsonResp{Code: status, Message: errMsg})
		return
	}
	writeJSON(w, 200, jsonResp{Code: 0, Message: "ok", Data: res})
}
