package ratelimit

import (
	"errors"
	"sync"
	"time"
)

// ErrRateLimited is returned by Allow when the window limit has been reached.
var ErrRateLimited = errors.New("rate limit exceeded")

// Limiter enforces "at most limit requests per rolling window", safe for
// concurrent use. A limit <= 0 disables limiting entirely.
type Limiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	requests []time.Time // timestamps of accepted requests, oldest first
}

// New creates a limiter that admits at most `limit` requests per rolling window.
func New(limit int, window time.Duration) *Limiter {
	return &Limiter{limit: limit, window: window}
}

// Allow reports whether a request may proceed now.
func (l *Limiter) Allow() error {
	if l.limit <= 0 {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	// Evict timestamps that have fallen out of the window.
	i := 0
	for i < len(l.requests) && !l.requests[i].After(cutoff) {
		i++
	}
	if i > 0 {
		l.requests = l.requests[i:]
	}

	if len(l.requests) >= l.limit {
		return ErrRateLimited
	}
	l.requests = append(l.requests, now)
	return nil
}
