package strategy

import (
	"sync"
	"time"
)

// SlidingCounter implements a sliding-window counter using fixed sub-windows (buckets).
// It divides the window into NumBuckets and keeps a counter per bucket. The total
// across buckets is used to decide allowance.
type SlidingCounter struct {
	Limit      int
	Window     time.Duration
	NumBuckets int

	mu           sync.Mutex
	buckets      []int
	currentIndex int
	lastTick     time.Time
}

func (sc *SlidingCounter) ensureInitialized(now time.Time) {
	if sc.NumBuckets <= 0 {
		sc.NumBuckets = 10
	}
	if sc.buckets == nil || len(sc.buckets) != sc.NumBuckets {
		sc.buckets = make([]int, sc.NumBuckets)
		sc.currentIndex = 0
		sc.lastTick = now
	}
}

// advanceBuckets moves the current index forward depending on elapsed time
// and clears buckets that are now outside the window.
func (sc *SlidingCounter) advanceBuckets(now time.Time) {
	bucketDuration := sc.Window / time.Duration(sc.NumBuckets)
	if bucketDuration <= 0 {
		return
	}
	elapsed := now.Sub(sc.lastTick)
	steps := int(elapsed / bucketDuration)
	if steps <= 0 {
		return
	}
	if steps >= sc.NumBuckets {
		// too much time passed; reset all
		for i := range sc.buckets {
			sc.buckets[i] = 0
		}
		sc.currentIndex = 0
	} else {
		for i := 0; i < steps; i++ {
			sc.currentIndex = (sc.currentIndex + 1) % sc.NumBuckets
			sc.buckets[sc.currentIndex] = 0
		}
	}
	sc.lastTick = sc.lastTick.Add(time.Duration(steps) * bucketDuration)
}

// Allow records an event in the current bucket and returns true if the
// total events in the sliding window is within the limit.
func (sc *SlidingCounter) Allow() bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	now := time.Now()
	sc.ensureInitialized(now)
	sc.advanceBuckets(now)

	// compute total
	total := 0
	for _, c := range sc.buckets {
		total += c
	}
	if total < sc.Limit {
		sc.buckets[sc.currentIndex]++
		return true
	}
	return false
}

type SlidingCounterMetrics struct {
	Limit        int   `json:"limit"`
	WindowSecs   int64 `json:"window_seconds"`
	NumBuckets   int   `json:"num_buckets"`
	CurrentCount int   `json:"current_count"`
}

func (sc *SlidingCounter) Snapshot() SlidingCounterMetrics {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	now := time.Now()
	sc.ensureInitialized(now)
	sc.advanceBuckets(now)

	total := 0
	for _, c := range sc.buckets {
		total += c
	}
	return SlidingCounterMetrics{
		Limit:        sc.Limit,
		WindowSecs:   int64(sc.Window / time.Second),
		NumBuckets:   sc.NumBuckets,
		CurrentCount: total,
	}
}
