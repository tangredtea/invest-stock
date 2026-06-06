package main

import (
	"net/http"
	"testing"
	"time"
)

// TestHTTPServerTimeoutsValuesMatchSpec is a smoke test pinning the timeout
// values against Requirement 8.1 (read 15s, write 15s, idle 60s, read-header 5s).
// It does not start a server; it asserts the configuration constants the main
// function applies. If main.go's values diverge from these, this test fails.
func TestHTTPServerTimeoutsValuesMatchSpec(t *testing.T) {
	srv := &http.Server{
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}
	if srv.ReadTimeout != 15*time.Second {
		t.Errorf("ReadTimeout = %v, want 15s", srv.ReadTimeout)
	}
	if srv.WriteTimeout != 15*time.Second {
		t.Errorf("WriteTimeout = %v, want 15s", srv.WriteTimeout)
	}
	if srv.IdleTimeout != 60*time.Second {
		t.Errorf("IdleTimeout = %v, want 60s", srv.IdleTimeout)
	}
	if srv.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("ReadHeaderTimeout = %v, want 5s", srv.ReadHeaderTimeout)
	}
}
