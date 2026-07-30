package data

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"
)

// roundTripFunc adapts a function to an http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// withTransport temporarily swaps httpClient's transport for the duration of fn.
func withTransport(rt http.RoundTripper, fn func()) {
	orig := httpClient.Transport
	httpClient.Transport = rt
	defer func() { httpClient.Transport = orig }()
	fn()
}

func withRetryDelay(delay func(int) time.Duration, fn func()) {
	orig := retryDelay
	retryDelay = delay
	defer func() { retryDelay = orig }()
	fn()
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

func okResponse(body string) *http.Response {
	return response(http.StatusOK, body)
}

// timeoutErr is a net.Error reporting a timeout, used to simulate a transient error.
type timeoutErr struct{}

func (timeoutErr) Error() string   { return "simulated timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

func TestHTTPClientTimeoutIs10s(t *testing.T) {
	// Feature: security-and-performance-hardening — Smoke for Requirement 7.1
	if httpClient.Timeout != 10*time.Second {
		t.Errorf("httpClient.Timeout = %v, want 10s", httpClient.Timeout)
	}
}

func TestMarketDataURLsEncodeSecID(t *testing.T) {
	malicious := "1.600000&fields=fake"
	for name, rawURL := range map[string]string{
		"kline": klineURL(malicious),
		"quote": quoteURL(malicious),
	} {
		t.Run(name, func(t *testing.T) {
			u, err := url.Parse(rawURL)
			if err != nil {
				t.Fatalf("parse url: %v", err)
			}
			values := u.Query()
			if got := values.Get("secid"); got != malicious {
				t.Fatalf("secid = %q, want %q", got, malicious)
			}
			if got := len(values["secid"]); got != 1 {
				t.Fatalf("secid param count = %d, want 1", got)
			}
			if name == "quote" {
				if got := values["fields"]; len(got) != 1 || got[0] != "f43,f44,f45,f46,f60" {
					t.Fatalf("fields params = %#v, want only canonical quote fields", got)
				}
			}
		})
	}
}

func TestNormalizeSecID(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{name: "shanghai", in: "1.600000", want: "1.600000", ok: true},
		{name: "shenzhen", in: "0.000001", want: "0.000001", ok: true},
		{name: "trims whitespace", in: " \t1.600519\n", want: "1.600519", ok: true},
		{name: "raw code rejected", in: "600000"},
		{name: "bad market", in: "2.600000"},
		{name: "non digit", in: "1.60x000"},
		{name: "query injection", in: "1.600000&fields=fake"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := normalizeSecID(tt.in)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("secid = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFetchRejectsInvalidSecIDBeforeNetwork(t *testing.T) {
	attempts := 0
	withTransport(roundTripFunc(func(*http.Request) (*http.Response, error) {
		attempts++
		return okResponse(`{"data":{}}`), nil
	}), func() {
		if _, err := FetchKLines("600000"); err == nil {
			t.Fatal("FetchKLines accepted raw stock code")
		}
		if _, err := FetchQuote("1.600000&fields=fake"); err == nil {
			t.Fatal("FetchQuote accepted injected secid")
		}
	})
	if attempts != 0 {
		t.Fatalf("invalid secid should not hit network; attempts = %d", attempts)
	}
}

func TestFetchKLinesNormalizesSecIDBeforeCacheLookup(t *testing.T) {
	SetKLineCacheTTL(time.Hour)
	SeedKLineCache("1.600000", []KLine{
		{Date: mustParseDay(t, "2023-12-08"), Open: 1, High: 1.2, Low: 0.9, Close: 1.1, Volume: 1000},
		{Date: mustParseDay(t, "2023-12-09"), Open: 1.1, High: 1.3, Low: 1, Close: 1.2, Volume: 1100},
	})

	attempts := 0
	withTransport(roundTripFunc(func(*http.Request) (*http.Response, error) {
		attempts++
		return okResponse(`{"data":{"klines":[]}}`), nil
	}), func() {
		got, err := FetchKLines(" \n1.600000\t")
		if err != nil {
			t.Fatalf("FetchKLines error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("cached kline length = %d, want 2", len(got))
		}
	})
	if attempts != 0 {
		t.Fatalf("trimmed secid should hit existing cache; network attempts = %d", attempts)
	}
}

func TestHTTPGetRetriesTransientThenSucceeds(t *testing.T) {
	// Requirement 7.3: transient errors are retried (up to 3 attempts total).
	attempts := 0
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		attempts++
		if attempts < 3 {
			return nil, timeoutErr{}
		}
		return okResponse("ok"), nil
	})
	withRetryDelay(func(int) time.Duration { return 0 }, func() {
		withTransport(rt, func() {
			body, err := httpGet("https://example.com/x")
			if err != nil {
				t.Fatalf("expected success on 3rd attempt, got error: %v", err)
			}
			if string(body) != "ok" {
				t.Errorf("unexpected body: %q", body)
			}
		})
	})
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestHTTPGetReturnsLastErrorAfterRetryLimit(t *testing.T) {
	// Requirements 7.3, 7.4: at most 3 attempts, then return the last error.
	attempts := 0
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		attempts++
		return nil, timeoutErr{}
	})
	withRetryDelay(func(int) time.Duration { return 0 }, func() {
		withTransport(rt, func() {
			if _, err := httpGet("https://example.com/x"); err == nil {
				t.Fatal("expected error after retry limit")
			}
		})
	})
	if attempts != 3 {
		t.Errorf("expected exactly 3 attempts, got %d", attempts)
	}
}

func TestHTTPGetDoesNotRetryNonRetryable(t *testing.T) {
	// Requirement 7.5: non-retryable errors return immediately (1 attempt).
	attempts := 0
	nonRetryable := errors.New("permanent failure")
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		attempts++
		return nil, nonRetryable
	})
	withTransport(rt, func() {
		if _, err := httpGet("https://example.com/x"); err == nil {
			t.Fatal("expected error for non-retryable failure")
		}
	})
	if attempts != 1 {
		t.Errorf("non-retryable error should not retry; got %d attempts", attempts)
	}
}

func TestHTTPGetRejectsNon2xxStatus(t *testing.T) {
	attempts := 0
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		attempts++
		return response(http.StatusNotFound, "not found"), nil
	})
	withTransport(rt, func() {
		if _, err := httpGet("https://example.com/missing"); err == nil {
			t.Fatal("expected error for non-2xx status")
		}
	})
	if attempts != 1 {
		t.Errorf("4xx status should not retry; got %d attempts", attempts)
	}
}

func TestHTTPGetRetriesServerErrorStatus(t *testing.T) {
	attempts := 0
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		attempts++
		if attempts < 3 {
			return response(http.StatusBadGateway, "bad gateway"), nil
		}
		return okResponse("ok"), nil
	})
	withRetryDelay(func(int) time.Duration { return 0 }, func() {
		withTransport(rt, func() {
			body, err := httpGet("https://example.com/flaky")
			if err != nil {
				t.Fatalf("expected success after server errors, got %v", err)
			}
			if string(body) != "ok" {
				t.Fatalf("body = %q, want ok", body)
			}
		})
	})
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestParseKLineRejectsMalformedNumbers(t *testing.T) {
	if _, ok := parseKLine("2023-12-08,0.979,bad,0.984,0.977,50000000"); ok {
		t.Fatal("expected malformed close to be rejected")
	}
	if _, ok := parseKLine("2023-12-08,0.979,0.980,0.984,0.977,-1"); ok {
		t.Fatal("expected negative volume to be rejected")
	}
	if _, ok := parseKLine("2023-12-08,0.979,0.980,0.970,0.977,50000000"); ok {
		t.Fatal("expected high below close to be rejected")
	}
	if _, ok := parseKLine("2023-12-08,0.979,0.980,0.984,0.981,50000000"); ok {
		t.Fatal("expected low above open to be rejected")
	}
	got, ok := parseKLine("2023-12-08,0.979,0.980,0.984,0.977,50000000")
	if !ok {
		t.Fatal("expected valid kline")
	}
	if got.Close != 0.980 || got.High != 0.984 || got.Low != 0.977 {
		t.Fatalf("parsed kline mismatch: %+v", got)
	}
}

func TestFetchKLinesRejectsInvalidSeriesAndDoesNotCache(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "malformed line",
			body: `{"data":{"klines":["2023-12-08,0.979,bad,0.984,0.977,50000000"]}}`,
		},
		{
			name: "duplicate day",
			body: `{"data":{"klines":["2023-12-08,0.979,0.980,0.984,0.977,50000000","2023-12-08,0.981,0.982,0.986,0.980,50000000"]}}`,
		},
		{
			name: "descending date",
			body: `{"data":{"klines":["2023-12-09,0.979,0.980,0.984,0.977,50000000","2023-12-08,0.981,0.982,0.986,0.980,50000000"]}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetKLineCacheTTL(time.Hour)
			attempts := 0
			withTransport(roundTripFunc(func(*http.Request) (*http.Response, error) {
				attempts++
				return okResponse(tt.body), nil
			}), func() {
				if _, err := FetchKLines("1.600000"); err == nil {
					t.Fatal("FetchKLines accepted invalid series")
				}
				if _, err := FetchKLines("1.600000"); err == nil {
					t.Fatal("FetchKLines accepted invalid series on second call")
				}
			})
			if attempts != 2 {
				t.Fatalf("invalid series should not be cached; attempts = %d, want 2", attempts)
			}
		})
	}
}

func TestValidateDailyKLinesRejectsSameDayBars(t *testing.T) {
	ks := []KLine{
		{Date: mustParseDay(t, "2023-12-08"), Open: 1, High: 1.1, Low: 0.9, Close: 1, Volume: 100},
		{Date: mustParseDay(t, "2023-12-08").Add(12 * time.Hour), Open: 1, High: 1.1, Low: 0.9, Close: 1, Volume: 100},
	}
	if err := ValidateDailyKLines("K线", ks); err == nil {
		t.Fatal("ValidateDailyKLines accepted duplicate same-day bars")
	}
}

func mustParseDay(t *testing.T, raw string) time.Time {
	t.Helper()
	day, err := time.Parse("2006-01-02", raw)
	if err != nil {
		t.Fatalf("parse day: %v", err)
	}
	return day
}

func TestFetchQuoteValidatesUpstreamData(t *testing.T) {
	tests := []struct {
		name string
		body string
		want Quote
		ok   bool
	}{
		{
			name: "valid full quote",
			body: `{"data":{"f43":1000,"f44":1100,"f45":900,"f46":950,"f60":980}}`,
			want: Quote{Price: 1, High: 1.1, Low: 0.9, Open: 0.95, PreClose: 0.98},
			ok:   true,
		},
		{
			name: "valid quote without intraday range",
			body: `{"data":{"f43":1000,"f44":0,"f45":0,"f46":0,"f60":980}}`,
			want: Quote{Price: 1, PreClose: 0.98},
			ok:   true,
		},
		{
			name: "zero price",
			body: `{"data":{"f43":0,"f44":1100,"f45":900,"f46":950,"f60":980}}`,
		},
		{
			name: "missing data",
			body: `{"data":null}`,
		},
		{
			name: "high below price",
			body: `{"data":{"f43":1000,"f44":900,"f45":800,"f46":850,"f60":980}}`,
		},
		{
			name: "low above price",
			body: `{"data":{"f43":1000,"f44":1100,"f45":1050,"f46":1080,"f60":980}}`,
		},
		{
			name: "high below low",
			body: `{"data":{"f43":1000,"f44":900,"f45":950,"f46":960,"f60":980}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTransport(roundTripFunc(func(*http.Request) (*http.Response, error) {
				return okResponse(tt.body), nil
			}), func() {
				got, err := FetchQuote("1.513630")
				if tt.ok {
					if err != nil {
						t.Fatalf("FetchQuote error: %v", err)
					}
					if got != tt.want {
						t.Fatalf("FetchQuote = %+v, want %+v", got, tt.want)
					}
					return
				}
				if err == nil {
					t.Fatalf("FetchQuote accepted invalid quote: %+v", got)
				}
			})
		})
	}
}
