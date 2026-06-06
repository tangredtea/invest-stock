package data

import (
	"bytes"
	"errors"
	"io"
	"net/http"
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

func okResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
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
	withTransport(rt, func() {
		body, err := httpGet("https://example.com/x")
		if err != nil {
			t.Fatalf("expected success on 3rd attempt, got error: %v", err)
		}
		if string(body) != "ok" {
			t.Errorf("unexpected body: %q", body)
		}
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
	withTransport(rt, func() {
		if _, err := httpGet("https://example.com/x"); err == nil {
			t.Fatal("expected error after retry limit")
		}
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
