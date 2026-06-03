package load_balancer

import (
	"errors"
	"sync"
)

type RoundRobin struct {
	instances []*Instance
	current   int
	mu        sync.Mutex
}

func NewRoundRobin(instances []*Instance) *RoundRobin {
	return &RoundRobin{
		instances: instances,
		current:   0,
	}
}

func (rr *RoundRobin) NextInstance() (*Instance, error) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	n := len(rr.instances)
	if n == 0 {
		return nil, errors.New("no instances available")
	}
	for i := 0; i < n; i++ {
		instance := rr.instances[rr.current]
		rr.current = (rr.current + 1) % n
		if instance != nil && instance.IsHealthy() {
			return instance, nil
		}
	}
	return nil, errors.New("no healthy instances available")
}
