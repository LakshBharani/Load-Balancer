package balancer

import (
	"hash/fnv"

	"github.com/LakshBharani/Load-Balancer/internal/backend"
)

// SourceIPHash always sends a given client IP to the same backend, as long
// as the pool composition doesn't change.
type SourceIPHash struct {
	pool *backend.Pool
}

func NewSourceIPHash(pool *backend.Pool) *SourceIPHash {
	return &SourceIPHash{pool: pool}
}

func (s *SourceIPHash) ChooseBackend(ctx ConnectionInfo) *backend.Backend {
	backends := s.pool.Backends
	if len(backends) == 0 {
		return nil
	}

	h := fnv.New64a()
	h.Write(ctx.ClientIP.AsSlice())
	idx := h.Sum64() % uint64(len(backends))
	return backends[idx]
}
