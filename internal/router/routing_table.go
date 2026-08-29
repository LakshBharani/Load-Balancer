package router

import (
	"net/netip"

	"github.com/LakshBharani/Load-Balancer/internal/backend"
	"github.com/LakshBharani/Load-Balancer/internal/balancer"
)

// Entry pairs a client-matching CIDR prefix with the index of the Balancer
// (in the owning RoutingTable) that should handle connections from it.
type Entry struct {
	Prefix      netip.Prefix
	BalancerIdx int
}

// RoutingTable is everything a single listening port needs to route an
// incoming connection: entries sorted most-specific-prefix-first (so the
// first match is the longest-prefix match), and the balancers they refer to.
type RoutingTable struct {
	Balancers []balancer.Balancer
	Entries   []Entry
}

// ChooseBackend finds the balancer for clientIP (first entry whose prefix
// contains it, since Entries is sorted most-specific-first) and asks it to
// pick a backend. Returns nil if no rule matches this client.
func (t *RoutingTable) ChooseBackend(clientIP netip.Addr) *backend.Backend {
	for _, e := range t.Entries {
		if e.Prefix.Contains(clientIP) {
			return t.Balancers[e.BalancerIdx].ChooseBackend(balancer.ConnectionInfo{ClientIP: clientIP})
		}
	}
	return nil
}
