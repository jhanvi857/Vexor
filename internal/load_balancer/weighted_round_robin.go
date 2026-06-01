package load_balancer

import (
	"errors"
	"sync"
)

type WeightedRoundRobin struct {
	instances []*Instance
	curr      []int
	total     int
	mu        sync.Mutex
}

func NewWeightedRoundRobin(instances []*Instance) *WeightedRoundRobin {
	wr := &WeightedRoundRobin{instances: instances}
	wr.curr = make([]int, len(instances))
	wr.total = 0
	for i, inst := range instances {
		if inst == nil {
			continue
		}
		w := inst.Weight
		if w <= 0 {
			w = 1
		}
		wr.total += w
		wr.curr[i] = 0
	}
	return wr
}

func (wr *WeightedRoundRobin) NextInstance() (*Instance, error) {
	wr.mu.Lock()
	defer wr.mu.Unlock()
	n := len(wr.instances)
	if n == 0 {
		return nil, errors.New("no instances available")
	}
	var bestIndex = -1
	for i := 0; i < n; i++ {
		inst := wr.instances[i]
		if inst == nil || !inst.Healthy {
			continue
		}
		wr.curr[i] += inst.Weight
		if bestIndex == -1 || wr.curr[i] > wr.curr[bestIndex] {
			bestIndex = i
		}
	}
	if bestIndex == -1 {
		return nil, errors.New("no healthy instances available")
	}
	wr.curr[bestIndex] -= wr.total
	return wr.instances[bestIndex], nil
}
