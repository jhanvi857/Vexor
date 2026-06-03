package load_balancer

import "sync"

type Instance struct {
	ID                string
	URL               string
	Weight            int
	ActiveConnections int64
	Healthy           bool
	mu                sync.RWMutex
}

func (i *Instance) SetHealthy(healthy bool) {
	i.mu.Lock()
	i.Healthy = healthy
	i.mu.Unlock()
}

func (i *Instance) IsHealthy() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.Healthy
}
