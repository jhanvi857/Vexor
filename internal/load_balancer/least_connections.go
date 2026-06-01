package load_balancer

import (
	"errors"
	"sync"
	"sync/atomic"
)

type LeastConnections struct {
	instances []*Instance
	mu        sync.Mutex
}

func NewLeastConnections(instances []*Instance) *LeastConnections {
	return &LeastConnections{instances: instances}
}

func (lc *LeastConnections) NextInstance() (*Instance, error) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	if len(lc.instances) == 0 {
		return nil, errors.New("no instances available")
	}
	var chosen *Instance
	var min int64 = -1
	for _, inst := range lc.instances {
		if !inst.Healthy {
			continue
		}
		ac := atomic.LoadInt64(&inst.ActiveConnections)
		if chosen == nil || ac < min || min == -1 {
			chosen = inst
			min = ac
		}
	}
	if chosen == nil {
		return nil, errors.New("no healthy instances available")
	}
	return chosen, nil
}
