package strategy

import (
	"math"
	"sync"
	"time"
)

type TokenBucket struct {
	Capacity       int
	Tokens         float64
	RefillRate     float64
	LastRefillTime time.Time
	mu             sync.Mutex
}

func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(tb.LastRefillTime).Seconds()
	newTokens := elapsed * tb.RefillRate
	tb.Tokens = math.Min(float64(tb.Capacity), tb.Tokens+newTokens)
	tb.LastRefillTime = now
	if tb.Tokens >= 1 {
		tb.Tokens -= 1
		return true
	}
	return false
}
