package main

import (
	"sync"

	"invest/pkg/data"
)

type dataSource struct {
	mu          sync.RWMutex
	fetchKLines func(string) ([]data.KLine, error)
	fetchQuote  func(string) (data.Quote, error)
}

var marketData = dataSource{
	fetchKLines: data.FetchKLines,
	fetchQuote:  data.FetchQuote,
}

func fetchKLines(secid string) ([]data.KLine, error) {
	marketData.mu.RLock()
	fn := marketData.fetchKLines
	marketData.mu.RUnlock()
	return fn(secid)
}

func fetchQuote(secid string) (data.Quote, error) {
	marketData.mu.RLock()
	fn := marketData.fetchQuote
	marketData.mu.RUnlock()
	return fn(secid)
}
