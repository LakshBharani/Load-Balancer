package balancer

import (
	"net/netip"

	"github.com/LakshBharani/Load-Balancer/internal/backend"
)

// ConnectionInfo carries per-connection context a Balancer may use to pick
// a backend (currently just the client's address).
type ConnectionInfo struct {
	ClientIP netip.Addr
}

// Balancer picks a backend for an incoming connection. Implementations are
// not required to be safe for concurrent use — the router gives every
// (rule, port) pair its own Balancer instance so a single goroutine per
// listener owns it exclusively.
type Balancer interface {
	ChooseBackend(ctx ConnectionInfo) *backend.Backend
}
