package observability

import "sync"

type SnapshotFunc func() interface{}

type Registry struct {
	mu    sync.RWMutex
	funcs map[string]SnapshotFunc
}

func NewRegistry() *Registry {
	return &Registry{funcs: make(map[string]SnapshotFunc)}
}

func (r *Registry) Register(name string, fn SnapshotFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.funcs[name] = fn
}

func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.funcs, name)
}

func (r *Registry) Snapshot() map[string]interface{} {
	r.mu.RLock()
	// copy keys to avoid holding lock while calling user funcs
	names := make([]string, 0, len(r.funcs))
	copied := make(map[string]SnapshotFunc, len(r.funcs))
	for k, v := range r.funcs {
		names = append(names, k)
		copied[k] = v
	}
	r.mu.RUnlock()

	out := make(map[string]interface{}, len(copied))
	for name, fn := range copied {
		// protect against panics in snapshot functions
		func() {
			defer func() { _ = recover() }()
			out[name] = fn()
		}()
	}
	return out
}
