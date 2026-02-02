package ratelimit

import (
	"context"
	"fmt"

	"golang.org/x/time/rate"
)

// Limiter wraps rate.Limiter with a name for logging/debugging.
type Limiter struct {
	limiter *rate.Limiter
	name    string
}

// New creates a new rate limiter with the given requests per second.
// The burst size equals the rate, allowing short bursts up to the rate limit.
func New(name string, requestsPerSecond int) *Limiter {
	return &Limiter{
		limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), requestsPerSecond),
		name:    name,
	}
}

// NewWithBurst creates a new rate limiter with custom burst size.
func NewWithBurst(name string, requestsPerSecond, burst int) *Limiter {
	return &Limiter{
		limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), burst),
		name:    name,
	}
}

// Wait blocks until the rate limiter allows a request to proceed.
// Returns an error if the context is cancelled.
func (l *Limiter) Wait(ctx context.Context) error {
	if err := l.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit wait for %s: %w", l.name, err)
	}
	return nil
}

// Allow reports whether a request can proceed without blocking.
// Use this for non-blocking checks; prefer Wait for most cases.
func (l *Limiter) Allow() bool {
	return l.limiter.Allow()
}

// Name returns the name of this rate limiter.
func (l *Limiter) Name() string {
	return l.name
}
