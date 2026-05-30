package breaker

import (
	"testing"
	"time"
)

func TestCircuitBreakerTransitions(t *testing.T) {
	cb := NewCircuitBreaker(2, 20*time.Millisecond)

	if !cb.AllowRequest() {
		t.Fatal("expected closed breaker to allow")
	}

	cb.RecordFailure()
	if cb.State != CLOSED {
		t.Fatalf("expected state to remain closed after first failure, got %v", cb.State)
	}

	cb.RecordFailure()
	if cb.State != OPEN {
		t.Fatalf("expected open after threshold failures, got %v", cb.State)
	}

	if cb.AllowRequest() {
		t.Fatal("expected open breaker to block before timeout")
	}

	time.Sleep(30 * time.Millisecond)
	if !cb.AllowRequest() {
		t.Fatal("expected open breaker to transition to half-open after timeout")
	}
	if cb.State != HALF_OPEN {
		t.Fatalf("expected half-open after timeout, got %v", cb.State)
	}

	cb.RecordSuccess()
	if cb.State != CLOSED {
		t.Fatalf("expected closed after success in half-open, got %v", cb.State)
	}

	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State != OPEN {
		t.Fatalf("expected open after threshold failures again, got %v", cb.State)
	}

	time.Sleep(30 * time.Millisecond)
	if !cb.AllowRequest() {
		t.Fatal("expected half-open after timeout for failure path")
	}
	if cb.State != HALF_OPEN {
		t.Fatalf("expected half-open before failure in probe, got %v", cb.State)
	}

	cb.RecordFailure()
	if cb.State != OPEN {
		t.Fatalf("expected open after failure in half-open, got %v", cb.State)
	}
}
