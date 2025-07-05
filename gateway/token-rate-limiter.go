package gateway

import (
	"sync"
	"time"
)

// Helper method to easily control ticking rate limits without having actual time ticker in background.
// All time related fields like regenTime or fullTimestamp are of uint64 type and represent number of milliseconds.
type TokenRateLimiter struct {
	mu            sync.Mutex
	regenTime     uint64 // Time in ms to regenerate one token
	fullTimestamp uint64 // Timestamp (ms) when max tokens are restored
	maxTokens     uint16
}

func (trl *TokenRateLimiter) AvailableTokens() uint16 {
	now := uint64(time.Now().UTC().UnixMilli())

	trl.mu.Lock()
	if now >= trl.fullTimestamp {
		trl.mu.Unlock()
		return trl.maxTokens
	}

	tokens := uint16((trl.fullTimestamp - uint64(time.Now().UTC().UnixMilli())) / trl.regenTime)

	trl.mu.Unlock()
	return tokens
}

func (trl *TokenRateLimiter) TryConsume() bool {
	tokens := trl.AvailableTokens()

	if tokens == 0 {
		return false
	}

	trl.mu.Lock()
	trl.fullTimestamp = uint64(time.Now().UTC().UnixMilli()) + trl.regenTime
	trl.mu.Unlock()
	return true
}

func NewTokenRateLimiter(tokenAmount uint16, regenTime time.Duration) *TokenRateLimiter {
	if tokenAmount == 0 {
		panic("token amount cannot be zero")
	}

	if regenTime == 0 {
		panic("token regeneration time cannot be zero")
	}

	return &TokenRateLimiter{
		regenTime:     uint64(regenTime.Milliseconds()),
		fullTimestamp: uint64(time.Now().UTC().UnixMilli()),
		maxTokens:     tokenAmount,
	}
}
