package ratelimit

import (
	"context"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	rate       float64
	bucketSize float64
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter with the specified rate and bucket size
func NewRateLimiter(rate float64, bucketSize float64) *RateLimiter {
	return &RateLimiter{
		rate:       rate,
		bucketSize: bucketSize,
		tokens:     bucketSize,
		lastRefill: time.Now(),
	}
}

// Wait blocks until a token is available or the context is cancelled
func (rl *RateLimiter) Wait(ctx context.Context) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	rl.tokens = min(rl.bucketSize, rl.tokens+elapsed*rl.rate)
	rl.lastRefill = now

	// If we have tokens, consume one and return
	if rl.tokens >= 1 {
		rl.tokens--
		return nil
	}

	// Calculate wait time
	waitTime := time.Duration((1 - rl.tokens) / rl.rate * float64(time.Second))

	// Wait for token or context cancellation
	select {
	case <-time.After(waitTime):
		rl.tokens = 0
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
