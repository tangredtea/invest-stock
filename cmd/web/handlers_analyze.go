package main

import (
	"net/http"

	"invest/internal/analysis"
	"invest/pkg/strategy"
)

func handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	loaded, clientErr, err := loadAnalyzeData(r.URL.Query().Get("code"))
	if msg := analyzeHTTPErrorMessage(clientErr); msg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	q, err := fetchQuote(loaded.SecID)
	var quoteWarning string
	if err != nil {
		quoteWarning = "实时行情不可用: " + err.Error()
	}
	sig := analysis.DefaultCompositeSignal(loaded.Snapshot, q)

	bt := strategy.RunBacktest(loaded.Klines)

	writeJSON(w, http.StatusOK, analyzeResp{
		Code: loaded.Code, SecID: loaded.SecID,
		Klines: loaded.Klines, Indicators: loaded.Snapshot.Series, Current: loaded.Snapshot.Current,
		Signal: sig, Backtest: bt, QuoteWarning: quoteWarning,
	})
}

func handleQuote(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	code := r.URL.Query().Get("code")
	secid, clientErr := resolveSecIDForCode(code)
	if clientErr != klineClientErrorNone {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效股票代码"})
		return
	}
	q, err := fetchQuote(secid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, q)
}
