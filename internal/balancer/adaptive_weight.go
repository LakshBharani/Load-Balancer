package balancer

import (
	"math"
	"math/rand/v2"

	"github.com/LakshBharani/Load-Balancer/internal/backend"
)

// AdaptiveWeight balances by composite server load (CPU/mem/net/io, pushed
// over the health side-channel) versus a per-backend weight that adapts
// over time. Ported from the design in
// https://www.wcse.org/WCSE_2018/W110.pdf.
type AdaptiveWeight struct {
	pool         []*adaptiveNode
	coefficients [4]float64
	alpha        float64
}

type adaptiveNode struct {
	backend *backend.Backend
	weight  float64
}

func NewAdaptiveWeight(pool *backend.Pool, coefficients [4]float64, alpha float64) *AdaptiveWeight {
	nodes := make([]*adaptiveNode, len(pool.Backends))
	for i, b := range pool.Backends {
		nodes[i] = &adaptiveNode{backend: b, weight: 1}
	}
	return &AdaptiveWeight{pool: nodes, coefficients: coefficients, alpha: alpha}
}

func (a *AdaptiveWeight) compositeLoad(b *backend.Backend) float64 {
	cpu, mem, net, io := b.Metrics.Snapshot()
	c := a.coefficients
	return c[0]*cpu + c[1]*mem + c[2]*net + c[3]*io
}

func (a *AdaptiveWeight) ChooseBackend(_ ConnectionInfo) *backend.Backend {
	if len(a.pool) == 0 {
		return nil
	}

	// Compute remaining capacity R_i = 100 - composite_load, aggregated.
	var rSum, wSum float64
	var lSum int64
	for _, node := range a.pool {
		rSum += a.compositeLoad(node.backend)
		wSum += node.weight
		lSum += node.backend.ActiveConnections()
	}

	safeWSum := math.Max(wSum, 1e-12)
	threshold := a.alpha * (rSum / safeWSum)

	for _, node := range a.pool {
		if node.weight <= 0.001 {
			continue
		}

		risk := a.compositeLoad(node.backend)
		ratio := risk / node.weight
		if ratio <= threshold {
			return node.backend
		}
	}

	// No server satisfied Ri/Wi <= threshold, meaning every server is
	// relatively overloaded — adjust weights (formula 6) and pick the
	// backend with the smallest current load.
	lSumF64 := float64(lSum)

	var totalLwi float64
	for _, node := range a.pool {
		load := float64(node.backend.ActiveConnections())
		weight := math.Max(node.weight, 1e-12)
		totalLwi += load * (safeWSum / weight) * lSumF64
	}
	avgLwi := math.Max(totalLwi/float64(len(a.pool)), 1e-12)

	var best *backend.Backend
	minLoad := int64(math.MaxInt64)

	for _, node := range a.pool {
		load := node.backend.ActiveConnections()
		loadF64 := float64(load)
		weight := math.Max(node.weight, 1e-12)

		lwi := loadF64 * (safeWSum / weight) * lSumF64
		adj := 1.0 - (lwi / avgLwi)
		node.weight += adj
		node.weight = clamp(node.weight, 0.1, 100.0)

		if load < minLoad {
			minLoad = load
			best = node.backend
		}
	}

	if best != nil {
		return best
	}
	return a.pool[rand.IntN(len(a.pool))].backend
}

func clamp(v, lo, hi float64) float64 {
	return math.Min(math.Max(v, lo), hi)
}
