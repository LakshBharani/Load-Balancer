package backend

import "sync"

// Metrics holds the last-reported resource usage for a physical server.
// Multiple Backends (different ports on the same host) can share one
// Metrics instance, since it's keyed by IP, not by backend id.
type Metrics struct {
	mu  sync.RWMutex
	CPU float64
	Mem float64
	Net float64
	IO  float64
}

func (m *Metrics) Update(cpu, mem, net, io float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CPU, m.Mem, m.Net, m.IO = cpu, mem, net, io
}

func (m *Metrics) Snapshot() (cpu, mem, net, io float64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.CPU, m.Mem, m.Net, m.IO
}
