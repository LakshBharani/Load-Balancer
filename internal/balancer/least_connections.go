package balancer

import (
	"github.com/LakshBharani/Load-Balancer/internal/backend"
)

// LeastConnections picks the backend with the fewest active proxied
// connections. Ties broken by pool order.
//
// Note: the reference implementation this was ported from advertised this
// strategy but shipped it as an empty stub — this is a real implementation.
type LeastConnections struct {
	pool *backend.Pool
}

func NewLeastConnections(pool *backend.Pool) *LeastConnections {
	return &LeastConnections{pool: pool}
}

func (l *LeastConnections) ChooseBackend(_ ConnectionInfo) *backend.Backend {
	backends := l.pool.Backends
	if len(backends) == 0 {
		return nil
	}

	best := backends[0]
	bestLoad := best.ActiveConnections()

	for _, b := range backends[1:] {
		if load := b.ActiveConnections(); load < bestLoad {
			best, bestLoad = b, load
		}
	}

	return best
}
