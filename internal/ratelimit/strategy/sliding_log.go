package strategy

import (
	"sync"
	"time"
)

// SlidingLog implements the sliding log rate limiting algorithm.
// It keeps timestamps of recent requests and allows a request
// if the number of timestamps within the window is below the limit.
type SlidingLog struct {
	Limit  int
	Window time.Duration
	mu     sync.Mutex
	logs   []time.Time
}

// Allow records the current timestamp and returns true if the
// number of requests in the rolling window is within the limit.
func (s *SlidingLog) Allow() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-s.Window)

	// drop old entries
	i := 0
	for ; i < len(s.logs); i++ {
		if s.logs[i].After(cutoff) {
			break
		}
	}
	if i > 0 {
		// keep only recent entries
		s.logs = append([]time.Time(nil), s.logs[i:]...)
	}

	if len(s.logs) < s.Limit {
		s.logs = append(s.logs, now)
		return true
	}
	return false
}

type SlidingLogMetrics struct {
	Limit        int   `json:"limit"`
	WindowSecs   int64 `json:"window_seconds"`
	CurrentCount int   `json:"current_count"`
}

func (s *SlidingLog) Snapshot() SlidingLogMetrics {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-s.Window)

	i := 0
	for ; i < len(s.logs); i++ {
		if s.logs[i].After(cutoff) {
			break
		}
	}
	if i > 0 {
		s.logs = append([]time.Time(nil), s.logs[i:]...)
	}

	return SlidingLogMetrics{
		Limit:        s.Limit,
		WindowSecs:   int64(s.Window / time.Second),
		CurrentCount: len(s.logs),
	}
}
