package strategy

import (
	"sync"
	"time"
)

type FixedWindow struct {
	Limit       int
	Count       int
	WindowStart time.Time
	mu          sync.Mutex
}

func (fw *FixedWindow) Allow() bool {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	now := time.Now()
	if now.Sub(fw.WindowStart) >= time.Minute {
		fw.Count = 0
		fw.WindowStart = now
	}
	fw.Count++
	return fw.Count <= fw.Limit
}
