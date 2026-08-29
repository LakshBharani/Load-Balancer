package config

import (
	"fmt"
	"log"
	"net/netip"
	"sort"
	"strconv"
	"strings"

	"github.com/LakshBharani/Load-Balancer/internal/backend"
	"github.com/LakshBharani/Load-Balancer/internal/balancer"
	"github.com/LakshBharani/Load-Balancer/internal/router"
)

// BuildResult is everything derived from an AppConfig: one RoutingTable per
// listening port, and the per-host Metrics the health listener should feed.
type BuildResult struct {
	Listeners map[uint16]*router.RoutingTable
	Healths   map[netip.Addr]*backend.Metrics
}

// parseClient splits "10.0.0.0/24:8080" into (10.0.0.0/24, 8080).
func parseClient(s string) (netip.Prefix, uint16, error) {
	idx := strings.LastIndex(s, ":")
	if idx < 0 {
		return netip.Prefix{}, 0, fmt.Errorf("badly formatted client: %s", s)
	}

	ipPart, portPart := s[:idx], s[idx+1:]

	port, err := strconv.ParseUint(portPart, 10, 16)
	if err != nil {
		return netip.Prefix{}, 0, fmt.Errorf("bad port: %s", s)
	}

	prefix, err := netip.ParsePrefix(ipPart)
	if err != nil {
		return netip.Prefix{}, 0, fmt.Errorf("bad ip/mask: %s", s)
	}

	return prefix, uint16(port), nil
}

// Build resolves an AppConfig into routing tables ready to serve traffic.
func Build(cfg *AppConfig) (*BuildResult, error) {
	healths := make(map[netip.Addr]*backend.Metrics)
	backends := make(map[string]*backend.Backend, len(cfg.Backends))

	for _, bc := range cfg.Backends {
		addr, err := netip.ParseAddrPort(bc.IP)
		if err != nil {
			return nil, fmt.Errorf("bad ip: %s", bc.IP)
		}
		ip := addr.Addr()

		m, ok := healths[ip]
		if !ok {
			m = &backend.Metrics{}
			healths[ip] = m
		}

		backends[bc.ID] = backend.New(bc.ID, addr, m)
	}

	listeners := make(map[uint16]*router.RoutingTable)

	for _, rule := range cfg.Rules {
		var targetBackends []*backend.Backend

		for _, targetName := range rule.Targets {
			if members, ok := cfg.Clusters[targetName]; ok {
				for _, memberID := range members {
					if b, ok := backends[memberID]; ok {
						targetBackends = append(targetBackends, b)
					}
				}
			} else if b, ok := backends[targetName]; ok {
				targetBackends = append(targetBackends, b)
			} else {
				log.Printf("warning: target %s not found", targetName)
			}
		}

		targetBackends = dedupeByID(targetBackends)
		if len(targetBackends) == 0 {
			log.Printf("warning: rule has no valid targets, skipping")
			continue
		}

		// Each distinct client port on this rule gets its own Balancer
		// instance, since a Balancer isn't safe for concurrent use and
		// each port is served by its own listener goroutine.
		portGroups := make(map[uint16][]netip.Prefix)
		for _, clientDef := range rule.Clients {
			prefix, port, err := parseClient(clientDef)
			if err != nil {
				return nil, err
			}
			portGroups[port] = append(portGroups[port], prefix)
		}

		for port, prefixes := range portGroups {
			table, ok := listeners[port]
			if !ok {
				table = &router.RoutingTable{}
				listeners[port] = table
			}

			pool := backend.NewPool(append([]*backend.Backend(nil), targetBackends...))

			bal, err := newBalancer(rule.Strategy, pool)
			if err != nil {
				return nil, err
			}

			balancerIdx := len(table.Balancers)
			table.Balancers = append(table.Balancers, bal)

			for _, prefix := range prefixes {
				table.Entries = append(table.Entries, router.Entry{Prefix: prefix, BalancerIdx: balancerIdx})
			}
		}
	}

	// Sort so the first matching entry is the longest-prefix (most
	// specific) match.
	for _, table := range listeners {
		sort.Slice(table.Entries, func(i, j int) bool {
			return table.Entries[i].Prefix.Bits() > table.Entries[j].Prefix.Bits()
		})
	}

	return &BuildResult{Listeners: listeners, Healths: healths}, nil
}

func newBalancer(s Strategy, pool *backend.Pool) (balancer.Balancer, error) {
	switch s.Type {
	case StrategyRoundRobin:
		return balancer.NewRoundRobin(pool), nil
	case StrategySourceIPHash:
		return balancer.NewSourceIPHash(pool), nil
	case StrategyLeastConnections:
		return balancer.NewLeastConnections(pool), nil
	case StrategyAdaptive:
		return balancer.NewAdaptiveWeight(pool, s.Coefficients, s.Alpha), nil
	default:
		return nil, fmt.Errorf("unknown strategy type: %q", s.Type)
	}
}

func dedupeByID(backends []*backend.Backend) []*backend.Backend {
	seen := make(map[string]bool, len(backends))
	out := backends[:0]
	for _, b := range backends {
		if !seen[b.ID] {
			seen[b.ID] = true
			out = append(out, b)
		}
	}
	return out
}
