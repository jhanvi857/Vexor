package breaker

import (
	"sync"
	"time"
)

type CircuitBreaker struct {
	State            State
	FailureCount     int
	FailureThreshold int
	ResetTimeout     time.Duration
	LastFailureTime  time.Time
	mu               sync.Mutex
}

func NewCircuitBreaker(failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		State:            CLOSED,
		FailureThreshold: failureThreshold,
		ResetTimeout:     resetTimeout,
	}
}

// AllowRequest : returns true when a request is allowed to proceed.
// transitions from OPEN to HALF_OPEN when the reset timeout has elapsed.
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.State {
	case OPEN:
		if time.Since(cb.LastFailureTime) >= cb.ResetTimeout {
			cb.State = HALF_OPEN
			return true
		}
		return false
	case HALF_OPEN:
		return true
	default:
		return true
	}
}

// RecordFailure : records a failed call. When failures reach the threshold
// the breaker moves to OPEN and the last failure time is recorded.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.FailureCount++
	cb.LastFailureTime = time.Now()

	if cb.State == HALF_OPEN || (cb.FailureThreshold > 0 && cb.FailureCount >= cb.FailureThreshold) {
		cb.State = OPEN
		cb.FailureCount = 0
	}
}

// RecordSuccess : records a successful call. On success the breaker resets
// to CLOSED and clears the failure count.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.FailureCount = 0
	cb.State = CLOSED
}
