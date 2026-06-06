package main

import (
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	defaultLoginThreshold = 5
	defaultLoginWindow    = 900 * time.Second
)

type ipCounter struct {
	count       int
	windowStart time.Time
}

// LoginRateLimiter throttles failed login attempts per client IP using a
// fixed-window failure counter. It is concurrency-safe.
type LoginRateLimiter struct {
	mu        sync.Mutex
	threshold int
	window    time.Duration
	attempts  map[string]*ipCounter
	now       func() time.Time // injectable clock for deterministic tests
}

// NewLoginRateLimiter creates a limiter with the given threshold and window,
// clamped to [1,100] and [60s,86400s] respectively.
func NewLoginRateLimiter(threshold int, window time.Duration) *LoginRateLimiter {
	if threshold < 1 {
		threshold = 1
	}
	if threshold > 100 {
		threshold = 100
	}
	if window < 60*time.Second {
		window = 60 * time.Second
	}
	if window > 86400*time.Second {
		window = 86400 * time.Second
	}
	return &LoginRateLimiter{
		threshold: threshold,
		window:    window,
		attempts:  make(map[string]*ipCounter),
		now:       time.Now,
	}
}

// Allowed reports whether a login attempt from ip may proceed. An expired
// window resets the counter and allows the attempt.
func (l *LoginRateLimiter) Allowed(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	c, ok := l.attempts[ip]
	if !ok {
		return true
	}
	if !l.now().Before(c.windowStart.Add(l.window)) {
		// Window elapsed: reset and allow.
		delete(l.attempts, ip)
		return true
	}
	return c.count < l.threshold
}

// RecordFailure increments the failure count for ip within the current window.
func (l *LoginRateLimiter) RecordFailure(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	c, ok := l.attempts[ip]
	if !ok || !now.Before(c.windowStart.Add(l.window)) {
		l.attempts[ip] = &ipCounter{count: 1, windowStart: now}
		return
	}
	c.count++
}

// Reset clears the failure count for ip (called on successful login).
func (l *LoginRateLimiter) Reset(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, ip)
}

// clientIP extracts the client IP from the request's RemoteAddr. X-Forwarded-For
// is intentionally not trusted by default to avoid spoofing.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
