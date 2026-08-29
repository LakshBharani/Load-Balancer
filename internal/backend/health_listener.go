package backend

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/netip"
	"sync/atomic"
)

type metricsPayload struct {
	CPU float64 `json:"cpu"`
	Mem float64 `json:"mem"`
	Net float64 `json:"net"`
	IO  float64 `json:"io"`
}

// HealthServer accepts TCP connections from backend hosts, each of which
// pushes newline-delimited JSON metrics for as long as it's connected.
// Metrics are attributed to a host by its connecting (source) IP.
//
// The map of per-host Metrics is held behind an atomic pointer so a config
// reload can swap in a freshly-built map (new Backend/Metrics instances)
// without needing to rebind the listening port.
type HealthServer struct {
	healths atomic.Pointer[map[netip.Addr]*Metrics]
}

func NewHealthServer(healths map[netip.Addr]*Metrics) *HealthServer {
	hs := &HealthServer{}
	hs.SetHealths(healths)
	return hs
}

func (hs *HealthServer) SetHealths(healths map[netip.Addr]*Metrics) {
	hs.healths.Store(&healths)
}

// Serve blocks, accepting connections until the listener is closed.
func (hs *HealthServer) Serve(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("health listener: %w", err)
	}
	log.Printf("healthcheck server listening on %s", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("health listener: accept error: %v", err)
			continue
		}
		go hs.handleMetricsStream(conn)
	}
}

func (hs *HealthServer) handleMetricsStream(conn net.Conn) {
	defer conn.Close()

	remote, ok := conn.RemoteAddr().(*net.TCPAddr)
	if !ok {
		return
	}
	peerIP, ok := netip.AddrFromSlice(remote.IP)
	if !ok {
		return
	}
	peerIP = peerIP.Unmap()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		if err := hs.processMetricsLine(peerIP, scanner.Bytes()); err != nil {
			log.Printf("health listener: skipping invalid packet from %s: %v", peerIP, err)
		}
	}
}

func (hs *HealthServer) processMetricsLine(peerIP netip.Addr, line []byte) error {
	var payload metricsPayload
	if err := json.Unmarshal(line, &payload); err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	healths := *hs.healths.Load()
	m, ok := healths[peerIP]
	if !ok {
		return fmt.Errorf("unknown server: %s", peerIP)
	}

	m.Update(payload.CPU, payload.Mem, payload.Net, payload.IO)
	return nil
}
