package balancer

import (
	"github.com/LakshBharani/Load-Balancer/internal/backend"
)

// RoundRobin cycles through the pool in order. Only the goroutine that
// owns this Balancer should call ChooseBackend.
type RoundRobin struct {
	pool  *backend.Pool
	index int
}

func NewRoundRobin(pool *backend.Pool) *RoundRobin {
	return &RoundRobin{pool: pool}
}

func (r *RoundRobin) ChooseBackend(_ ConnectionInfo) *backend.Backend {
	backends := r.pool.Backends
	if len(backends) == 0 {
		return nil
	}

	b := backends[r.index%len(backends)]
	r.index++
	return b
}
