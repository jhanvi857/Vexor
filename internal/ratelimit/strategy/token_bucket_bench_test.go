package strategy

import (
	"testing"
	"time"
)

func BenchmarkTokenBucketAllow(b *testing.B) {
	tb := &TokenBucket{
		Capacity:       1024,
		Tokens:         1024,
		RefillRate:     1024,
		LastRefillTime: time.Now(),
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = tb.Allow()
	}
}
