package data

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

// httpClient is a shared client with a finite timeout covering connection,
// request, and response-body read. Prevents a hung upstream from blocking
// goroutines indefinitely.
var httpClient = &http.Client{Timeout: 10 * time.Second}

var retryDelay = func(attempt int) time.Duration {
	return time.Duration(attempt) * time.Second
}

// isRetryable reports whether err is a transient error worth retrying
// (network connection errors, EOF while reading body, timeouts). Non-transient
// errors (e.g. a permanent failure or a JSON parse error handled by callers)
// are not retried here.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}
	// Timeouts (including those wrapped in *url.Error) are retryable.
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	// Connection-level network errors are retryable.
	var opErr *net.OpError
	return errors.As(err, &opErr)
}

// httpGet issues a GET through httpClient, retrying transient errors up to a
// total of 3 attempts with linear backoff (1s before attempt 2, 2s before 3).
func httpGet(url string) ([]byte, error) {
	var lastErr error
	for i := 0; i < 3; i++ {
		if i > 0 {
			time.Sleep(retryDelay(i))
		}
		resp, err := httpClient.Get(url)
		if err != nil {
			lastErr = err
			if isRetryable(err) {
				continue
			}
			return nil, err // non-retryable: return immediately
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			if isRetryable(err) {
				continue
			}
			return nil, err
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			lastErr = fmt.Errorf("HTTP状态异常: %d", resp.StatusCode)
			if resp.StatusCode >= http.StatusInternalServerError {
				continue
			}
			return nil, lastErr
		}
		return body, nil
	}
	return nil, lastErr
}
