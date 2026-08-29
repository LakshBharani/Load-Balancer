package router

import (
	"fmt"
	"log"
	"net"
	"net/netip"
	"sync/atomic"

	"github.com/LakshBharani/Load-Balancer/internal/proxy"
)

var nextConnID atomic.Uint64

// portListener owns one bound TCP port and routes accepted connections
// through whatever RoutingTable is currently swapped in, so config
// reloads can update routing without dropping the listener.
type portListener struct {
	port  uint16
	ln    net.Listener
	table atomic.Pointer[RoutingTable]
}

func newPortListener(port uint16, table *RoutingTable) (*portListener, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("bind port %d: %w", port, err)
	}

	pl := &portListener{port: port, ln: ln}
	pl.table.Store(table)
	return pl, nil
}

func (pl *portListener) update(table *RoutingTable) {
	pl.table.Store(table)
}

func (pl *portListener) stop() {
	pl.ln.Close()
}

func (pl *portListener) run() {
	log.Printf("listening on :%d", pl.port)
	for {
		conn, err := pl.ln.Accept()
		if err != nil {
			log.Printf("listener :%d stopped accepting: %v", pl.port, err)
			return
		}
		go pl.handle(conn)
	}
}

func (pl *portListener) handle(conn net.Conn) {
	remote, ok := conn.RemoteAddr().(*net.TCPAddr)
	if !ok {
		conn.Close()
		return
	}
	clientIP, ok := netip.AddrFromSlice(remote.IP)
	if !ok {
		conn.Close()
		return
	}
	clientIP = clientIP.Unmap()

	table := pl.table.Load()
	b := table.ChooseBackend(clientIP)
	if b == nil {
		log.Printf("error: no matching rule for %s on port %d", clientIP, pl.port)
		conn.Close()
		return
	}

	connID := nextConnID.Add(1)
	if err := proxy.TCPConnection(connID, conn, b); err != nil {
		log.Printf("error: conn_id=%d proxy failed: %v", connID, err)
	}
}

// Manager owns the set of active per-port listeners and lets a config
// reload reconcile them: existing ports get their routing table swapped
// in place, new ports get a fresh listener, and ports no longer present
// in config are stopped.
type Manager struct {
	listeners map[uint16]*portListener
}

func NewManager() *Manager {
	return &Manager{listeners: make(map[uint16]*portListener)}
}

// Reconcile applies a new set of routing tables (keyed by port) to the
// currently running listeners.
func (m *Manager) Reconcile(tables map[uint16]*RoutingTable) {
	for port, table := range tables {
		if pl, ok := m.listeners[port]; ok {
			pl.update(table)
			log.Printf("updated rules on port %d", port)
			continue
		}

		pl, err := newPortListener(port, table)
		if err != nil {
			log.Printf("error: %v", err)
			continue
		}
		m.listeners[port] = pl
		go pl.run()
	}

	for port, pl := range m.listeners {
		if _, ok := tables[port]; !ok {
			pl.stop()
			delete(m.listeners, port)
		}
	}
}
