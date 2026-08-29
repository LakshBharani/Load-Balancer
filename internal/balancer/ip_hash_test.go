package balancer

import (
	"fmt"
	"net/netip"
	"testing"

	"github.com/LakshBharani/Load-Balancer/internal/backend"
)

func dummyBackends(count int) *backend.Pool {
	backends := make([]*backend.Backend, count)
	for i := 0; i < count; i++ {
		backends[i] = newTestBackend(fmt.Sprintf("backend %d", i+1), "127.0.0.1", uint16(8081+i))
	}
	return backend.NewPool(backends)
}

func TestSourceIPHashSameIPSameBackend(t *testing.T) {
	s := NewSourceIPHash(dummyBackends(3))
	clientIP := netip.MustParseAddr("192.168.1.100")

	first := s.ChooseBackend(ConnectionInfo{ClientIP: clientIP})
	second := s.ChooseBackend(ConnectionInfo{ClientIP: clientIP})

	if first == nil || second == nil {
		t.Fatalf("expected non-nil picks, got %v, %v", first, second)
	}
	if first.ID != second.ID {
		t.Fatalf("same IP routed to different backends: %s vs %s", first.ID, second.ID)
	}
}

func TestSourceIPHashDistribution(t *testing.T) {
	pool := dummyBackends(3)
	s := NewSourceIPHash(pool)

	seen := map[string]bool{}
	for i := 0; i < 30; i++ {
		ip := netip.MustParseAddr(fmt.Sprintf("192.168.1.%d", 100+i))
		b := s.ChooseBackend(ConnectionInfo{ClientIP: ip})
		if b == nil {
			t.Fatalf("expected a backend for ip %s", ip)
		}
		seen[b.ID] = true
	}

	for _, b := range pool.Backends {
		if !seen[b.ID] {
			t.Errorf("backend %s received no traffic across 30 distinct IPs", b.ID)
		}
	}
}

func TestSourceIPHashEmptyPool(t *testing.T) {
	s := NewSourceIPHash(backend.NewPool(nil))
	if got := s.ChooseBackend(ConnectionInfo{ClientIP: netip.MustParseAddr("1.2.3.4")}); got != nil {
		t.Fatalf("want nil, got %v", got)
	}
}
