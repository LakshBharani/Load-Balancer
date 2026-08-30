package balancer

import (
	"fmt"
	"testing"

	"github.com/LakshBharani/Load-Balancer/internal/backend"
)

func newTestBackend(id, ip string, port uint16) *backend.Backend {
	return backend.New(id, fmt.Sprintf("%s:%d", ip, port), &backend.Metrics{})
}

func TestRoundRobinCyclesInOrder(t *testing.T) {
	pool := backend.NewPool([]*backend.Backend{
		newTestBackend("srv-0", "127.0.0.1", 3000),
		newTestBackend("srv-1", "127.0.0.1", 3001),
		newTestBackend("srv-2", "127.0.0.1", 3002),
	})
	rr := NewRoundRobin(pool)

	want := []string{"srv-0", "srv-1", "srv-2", "srv-0", "srv-1"}
	for i, w := range want {
		got := rr.ChooseBackend(ConnectionInfo{})
		if got.ID != w {
			t.Fatalf("pick %d: got %s, want %s", i, got.ID, w)
		}
	}
}

func TestRoundRobinEmptyPool(t *testing.T) {
	rr := NewRoundRobin(backend.NewPool(nil))
	if got := rr.ChooseBackend(ConnectionInfo{}); got != nil {
		t.Fatalf("want nil, got %v", got)
	}
}
