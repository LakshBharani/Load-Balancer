package balancer

import "testing"

func TestLeastConnectionsPicksSmallestLoad(t *testing.T) {
	pool := dummyBackends(3)
	pool.Backends[0].IncConnections()
	pool.Backends[0].IncConnections()
	pool.Backends[1].IncConnections()
	// pool.Backends[2] has 0 active connections.

	l := NewLeastConnections(pool)
	got := l.ChooseBackend(ConnectionInfo{})
	if got.ID != pool.Backends[2].ID {
		t.Fatalf("got %s, want %s", got.ID, pool.Backends[2].ID)
	}
}

func TestLeastConnectionsEmptyPool(t *testing.T) {
	l := NewLeastConnections(dummyBackends(0))
	if got := l.ChooseBackend(ConnectionInfo{}); got != nil {
		t.Fatalf("want nil, got %v", got)
	}
}
