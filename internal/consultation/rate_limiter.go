package consultation

import (
	"context"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	tokens         int
	maxTokens      int
	refillRate     time.Duration
	lastRefill     time.Time
	mutex          sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	refillRate := time.Minute / time.Duration(config.RequestsPerMinute)
	
	return &RateLimiter{
		tokens:     config.BurstSize,
		maxTokens:  config.BurstSize,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Wait waits for a token to become available
func (rl *RateLimiter) Wait(ctx context.Context) error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	tokensToAdd := int(elapsed / rl.refillRate)
	
	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}
		rl.lastRefill = now
	}

	// Check if token is available
	if rl.tokens > 0 {
		rl.tokens--
		return nil
	}

	// Calculate wait time for next token
	waitTime := rl.refillRate - (elapsed % rl.refillRate)
	
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitTime):
		rl.tokens = 0 // Will be refilled on next call
		return nil
	}
}

// TryAcquire attempts to acquire a token without waiting
func (rl *RateLimiter) TryAcquire() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	tokensToAdd := int(elapsed / rl.refillRate)
	
	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}
		rl.lastRefill = now
	}

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	
	return false
}

// GetAvailableTokens returns the current number of available tokens
func (rl *RateLimiter) GetAvailableTokens() int {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	tokensToAdd := int(elapsed / rl.refillRate)
	
	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}
		rl.lastRefill = now
	}
	
	return rl.tokens
}