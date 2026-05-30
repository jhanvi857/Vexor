package strategy

type TokenBucketMetrics struct {
	Tokens   float64 `json:"tokens"`
	Capacity int     `json:"capacity"`
}

func (tb *TokenBucket) Snapshot() TokenBucketMetrics {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	return TokenBucketMetrics{
		Tokens:   tb.Tokens,
		Capacity: tb.Capacity,
	}
}
