package strategy

import (
	"sync"
	"time"
)

type FixedWindow struct {
	Limit       int
	Window      time.Duration
	Count       int
	WindowStart time.Time
	mu          sync.Mutex
}

type FixedWindowMetrics struct {
	Limit       int       `json:"limit"`
	WindowSecs  int64     `json:"window_seconds"`
	Count       int       `json:"count"`
	WindowStart time.Time `json:"window_start"`
}

func (fw *FixedWindow) Allow() bool {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	now := time.Now()
	w := fw.Window
	if w <= 0 {
		w = time.Minute
	}
	if now.Sub(fw.WindowStart) >= w {
		fw.Count = 0
		fw.WindowStart = now
	}
	fw.Count++
	return fw.Count <= fw.Limit
}

func (fw *FixedWindow) Snapshot() FixedWindowMetrics {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	w := fw.Window
	if w <= 0 {
		w = time.Minute
	}
	return FixedWindowMetrics{
		Limit:       fw.Limit,
		WindowSecs:  int64(w / time.Second),
		Count:       fw.Count,
		WindowStart: fw.WindowStart,
	}
}
