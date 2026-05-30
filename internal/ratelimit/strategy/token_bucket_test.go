package strategy

import (
	"testing"
	"time"
)

func TestTokenBucketAllowsUpToCapacityThenDenies(t *testing.T) {
	tb := &TokenBucket{
		Capacity:       3,
		Tokens:         3,
		RefillRate:     0,
		LastRefillTime: time.Now(),
	}

	for i := 0; i < 3; i++ {
		if !tb.Allow() {
			t.Fatalf("expected request %d to be allowed", i+1)
		}
	}

	if tb.Allow() {
		t.Fatal("expected request past capacity to be denied")
	}
}

func TestTokenBucketRefillsOverTime(t *testing.T) {
	tb := &TokenBucket{
		Capacity:       2,
		Tokens:         0,
		RefillRate:     10,
		LastRefillTime: time.Now(),
	}

	if tb.Allow() {
		t.Fatal("expected empty bucket to deny before refill")
	}

	time.Sleep(150 * time.Millisecond)

	if !tb.Allow() {
		t.Fatal("expected bucket to allow after refill")
	}
}
