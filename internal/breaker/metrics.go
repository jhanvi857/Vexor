package breaker

import "time"

// Metrics holds a snapshot of a circuit breaker's state for reporting.
type Metrics struct {
	State            string    `json:"state"`
	FailureCount     int       `json:"failure_count"`
	FailureThreshold int       `json:"failure_threshold"`
	ResetTimeoutSecs int64     `json:"reset_timeout_seconds"`
	LastFailureTime  time.Time `json:"last_failure_time,omitempty"`
}

// Snapshot returns a copy of the current breaker metrics.
func Snapshot(cb *CircuitBreaker) Metrics {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return Metrics{
		State:            cb.State.String(),
		FailureCount:     cb.FailureCount,
		FailureThreshold: cb.FailureThreshold,
		ResetTimeoutSecs: int64(cb.ResetTimeout / time.Second),
		LastFailureTime:  cb.LastFailureTime,
	}
}
