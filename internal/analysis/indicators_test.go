package analysis

import (
	"reflect"
	"slices"
	"testing"

	"invest/pkg/data"
	"invest/pkg/indicator"
	"invest/pkg/strategy"
)

func TestLatestIndicatorsRequiresAtLeastTwoBars(t *testing.T) {
	if _, ok := LatestIndicators(nil); ok {
		t.Fatal("empty input should not produce a snapshot")
	}
	if _, ok := LatestIndicators([]data.KLine{{Close: 1}}); ok {
		t.Fatal("single-bar input should not produce a snapshot")
	}
}

func TestRequireLatestIndicatorsReturnsDataError(t *testing.T) {
	_, err := RequireLatestIndicators([]data.KLine{{Close: 1}})
	if err == nil {
		t.Fatal("expected error for single-bar input")
	}
	if got := err.Error(); got != "k线数量不足: 实际 1 根, 至少需要 2 根" {
		t.Fatalf("error = %q", got)
	}
}

func TestLatestIndicatorsReturnsCurrentAndPreviousFromOneSeries(t *testing.T) {
	klines := []data.KLine{
		{Close: 10},
		{Close: 11},
		{Close: 12},
		{Close: 13},
	}
	snapshot, ok := LatestIndicators(klines)
	if !ok {
		t.Fatal("expected snapshot")
	}

	closes := []float64{10, 11, 12, 13}
	series := indicator.ComputeSeries(closes)
	last := len(closes) - 1
	if !slices.Equal(snapshot.Closes, closes) {
		t.Fatalf("closes = %v, want %v", snapshot.Closes, closes)
	}
	if snapshot.Current != indicator.At(series, last) {
		t.Fatalf("current = %+v, want %+v", snapshot.Current, indicator.At(series, last))
	}
	if snapshot.Previous != indicator.At(series, last-1) {
		t.Fatalf("previous = %+v, want %+v", snapshot.Previous, indicator.At(series, last-1))
	}
	if snapshot.LastClose() != closes[last] {
		t.Fatalf("LastClose = %v, want %v", snapshot.LastClose(), closes[last])
	}
	if snapshot.LastIndex() != last {
		t.Fatalf("LastIndex = %v, want %v", snapshot.LastIndex(), last)
	}
}

func TestCompositeSignalMatchesStrategyComposite(t *testing.T) {
	klines := []data.KLine{
		{Close: 10},
		{Close: 11},
		{Close: 12},
		{Close: 13},
	}
	snapshot, ok := LatestIndicators(klines)
	if !ok {
		t.Fatal("expected snapshot")
	}
	quote := data.Quote{Price: 12.5, High: 13, Low: 12, PreClose: 12.2}
	holdings := 100000

	got := CompositeSignal(snapshot, quote, holdings)
	want := strategy.Composite(
		snapshot.LastClose(),
		snapshot.Current.MA5,
		snapshot.Current.MA20,
		snapshot.Current.MA60,
		snapshot.Current.RSI14,
		snapshot.Current.BollUpper,
		snapshot.Current.BollLower,
		snapshot.Current.MACDHist,
		snapshot.Previous.MACDHist,
		holdings,
		strategy.QuoteInfo{Price: quote.Price, High: quote.High, Low: quote.Low, PreClose: quote.PreClose},
	)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CompositeSignal = %+v, want %+v", got, want)
	}
}

func TestDefaultCompositeSignalUsesDefaultHoldings(t *testing.T) {
	klines := []data.KLine{
		{Close: 10},
		{Close: 11},
		{Close: 12},
	}
	snapshot, ok := LatestIndicators(klines)
	if !ok {
		t.Fatal("expected snapshot")
	}
	quote := data.Quote{Price: 11.5, High: 12, Low: 11, PreClose: 11.2}

	got := DefaultCompositeSignal(snapshot, quote)
	want := CompositeSignal(snapshot, quote, DefaultSignalHoldings)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DefaultCompositeSignal = %+v, want %+v", got, want)
	}
}
