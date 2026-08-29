package proxy

import (
	"io"
	"log"
	"net"
	"sync/atomic"
	"time"

	"github.com/LakshBharani/Load-Balancer/internal/backend"
)

// connectionContext tracks a single proxied connection's lifetime, mirroring
// the accounting the reference implementation does on connection close
// (active-connection counter, bytes transferred, duration).
type connectionContext struct {
	id        uint64
	client    net.Addr
	backend   *backend.Backend
	start     time.Time
	bytesSent atomic.Int64
}

func newConnectionContext(id uint64, client net.Addr, b *backend.Backend) *connectionContext {
	b.IncConnections()
	return &connectionContext{id: id, client: client, backend: b, start: time.Now()}
}

func (c *connectionContext) close() {
	c.backend.DecConnections()
	log.Printf(
		"info: conn_id=%d closed. client=%s backend=%s bytes=%d duration=%s",
		c.id, c.client, c.backend.Addr, c.bytesSent.Load(), time.Since(c.start),
	)
}

// TCPConnection dials the chosen backend and proxies bytes bidirectionally
// between it and the already-accepted client connection until either side
// closes.
func TCPConnection(connID uint64, clientConn net.Conn, b *backend.Backend) error {
	defer clientConn.Close()

	ctx := newConnectionContext(connID, clientConn.RemoteAddr(), b)
	defer ctx.close()

	backendConn, err := net.Dial("tcp", b.Addr.String())
	if err != nil {
		return err
	}
	defer backendConn.Close()

	errc := make(chan error, 2)

	go func() {
		n, err := io.Copy(backendConn, clientConn)
		ctx.bytesSent.Add(n)
		if tc, ok := backendConn.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
		errc <- err
	}()
	go func() {
		n, err := io.Copy(clientConn, backendConn)
		ctx.bytesSent.Add(n)
		if tc, ok := clientConn.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
		errc <- err
	}()

	err1 := <-errc
	err2 := <-errc
	if err1 != nil {
		return err1
	}
	return err2
}
