package backend

import (
	"fmt"
	"sync/atomic"
)

// Backend is a single proxied endpoint. Multiple Backends may share the
// same *Metrics (same physical host, different ports), hence it's a
// pointer field rather than owned.
//
// Addr is a dial address ("host:port" or "ip:port") rather than a parsed
// netip.AddrPort so backends can be addressed by hostname (Docker/k8s
// service names, host.docker.internal, etc.) — resolution happens per
// dial via net.Dial, same as any other Go TCP client.
type Backend struct {
	ID                string
	Addr              string
	Metrics           *Metrics
	activeConnections atomic.Int64
}

func New(id string, addr string, metrics *Metrics) *Backend {
	return &Backend{ID: id, Addr: addr, Metrics: metrics}
}

func (b *Backend) IncConnections() int64 {
	return b.activeConnections.Add(1)
}

func (b *Backend) DecConnections() int64 {
	return b.activeConnections.Add(-1)
}

func (b *Backend) ActiveConnections() int64 {
	return b.activeConnections.Load()
}

func (b *Backend) String() string {
	return fmt.Sprintf("%s (%s)", b.Addr, b.ID)
}

// Pool is a set of backends a single Balancer instance can choose among.
// Backend pointers may be shared across pools/balancers (e.g. a backend
// belonging to two clusters), so a Pool never mutates a Backend's identity.
type Pool struct {
	Backends []*Backend
}

func NewPool(backends []*Backend) *Pool {
	return &Pool{Backends: backends}
}
