package balancer

import (
	"testing"

	"github.com/LakshBharani/Load-Balancer/internal/backend"
)

func TestAdaptiveWeightChoosesLessLoadedHost(t *testing.T) {
	pool := dummyBackends(2)
	pool.Backends[0].Metrics.Update(90.0, 80.0, 10.0, 5.0)
	pool.Backends[1].Metrics.Update(10.0, 5.0, 1.0, 1.0)

	a := NewAdaptiveWeight(pool, [4]float64{0.5, 0.2, 0.2, 0.1}, 0.5)
	got := a.ChooseBackend(ConnectionInfo{})
	if got == nil || got.ID != pool.Backends[1].ID {
		t.Fatalf("got %v, want %s", got, pool.Backends[1].ID)
	}
}

func TestAdaptiveWeightEmptyPool(t *testing.T) {
	a := NewAdaptiveWeight(backend.NewPool(nil), [4]float64{0.5, 0.2, 0.2, 0.1}, 0.5)
	if got := a.ChooseBackend(ConnectionInfo{}); got != nil {
		t.Fatalf("want nil, got %v", got)
	}
}

func TestAdaptiveWeightRatioTriggersImmediateSelection(t *testing.T) {
	// threshold = alpha * (r_sum / w_sum) = 1.0 * (100 / 2) = 50.
	// server-0 ratio = 0 / 1 = 0 <= 50, so it's picked immediately.
	pool := dummyBackends(2)
	pool.Backends[0].Metrics.Update(0, 0, 0, 0)
	pool.Backends[1].Metrics.Update(100, 100, 100, 100)

	a := NewAdaptiveWeight(pool, [4]float64{0.25, 0.25, 0.25, 0.25}, 1.0)
	got := a.ChooseBackend(ConnectionInfo{})
	if got == nil || got.ID != pool.Backends[0].ID {
		t.Fatalf("got %v, want %s", got, pool.Backends[0].ID)
	}
}

func TestAdaptiveWeightFallsBackToMinLoad(t *testing.T) {
	// Using only the CPU coefficient: composite loads are 10, 5, 20.
	// threshold = 0.5 * (35/3) ~= 5.83; server-0's ratio (10) exceeds it,
	// server-1's ratio (5) doesn't, so server-1 is picked via the
	// immediate-selection path.
	pool := dummyBackends(3)
	pool.Backends[0].Metrics.Update(10, 10, 10, 10)
	pool.Backends[1].Metrics.Update(5, 5, 5, 5)
	pool.Backends[2].Metrics.Update(20, 20, 20, 20)

	a := NewAdaptiveWeight(pool, [4]float64{1.0, 0, 0, 0}, 0.5)
	got := a.ChooseBackend(ConnectionInfo{})
	if got == nil || got.ID != pool.Backends[1].ID {
		t.Fatalf("got %v, want %s", got, pool.Backends[1].ID)
	}
}
