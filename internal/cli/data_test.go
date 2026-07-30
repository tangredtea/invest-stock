package cli

import (
	"errors"
	"strings"
	"testing"
)

func TestFetchKLinesWrapsInvalidSecIDError(t *testing.T) {
	_, err := FetchKLines("bad")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "获取K线失败:") {
		t.Fatalf("error = %q, want wrapped K-line message", err.Error())
	}
}

func TestFetchQuoteWrapsInvalidSecIDError(t *testing.T) {
	_, err := FetchQuote("bad")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "获取实时行情失败:") {
		t.Fatalf("error = %q, want wrapped quote message", err.Error())
	}
}

func TestFetchKLinesPreservesUnderlyingError(t *testing.T) {
	err := errors.New("upstream unavailable")
	wrapped := wrapKLineError(err)
	if !errors.Is(wrapped, err) {
		t.Fatalf("wrapped error does not preserve source: %v", wrapped)
	}
}
