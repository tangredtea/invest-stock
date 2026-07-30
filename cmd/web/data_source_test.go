package main

import (
	"sync"
	"testing"

	"invest/pkg/data"
)

func stubFetchQuote(t *testing.T, fn func(string) (data.Quote, error)) {
	t.Helper()
	marketData.mu.Lock()
	orig := marketData.fetchQuote
	marketData.fetchQuote = fn
	marketData.mu.Unlock()
	t.Cleanup(func() {
		marketData.mu.Lock()
		marketData.fetchQuote = orig
		marketData.mu.Unlock()
	})
}

func stubFetchKLines(t *testing.T, fn func(string) ([]data.KLine, error)) {
	t.Helper()
	marketData.mu.Lock()
	orig := marketData.fetchKLines
	marketData.fetchKLines = fn
	marketData.mu.Unlock()
	t.Cleanup(func() {
		marketData.mu.Lock()
		marketData.fetchKLines = orig
		marketData.mu.Unlock()
	})
}

func TestDataSourceCanBeReadWhileStubsChange(t *testing.T) {
	stubFetchQuote(t, func(string) (data.Quote, error) {
		return data.Quote{Price: 1, PreClose: 1}, nil
	})
	stubFetchKLines(t, func(string) ([]data.KLine, error) {
		return nil, nil
	})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			setFetchQuoteForTest(func(string) (data.Quote, error) {
				return data.Quote{Price: float64(i + 1), PreClose: 1}, nil
			})
		}(i)
		go func() {
			defer wg.Done()
			_, _ = fetchQuote("1.600000")
			_, _ = fetchKLines("1.600000")
		}()
	}
	wg.Wait()
}

func setFetchQuoteForTest(fn func(string) (data.Quote, error)) {
	marketData.mu.Lock()
	marketData.fetchQuote = fn
	marketData.mu.Unlock()
}
