// Command connstress opens N simultaneous TCP
// connections through the load balancer, sends one request on each, holds
// briefly, then closes. Reports how many succeeded, to substantiate a
// "handles N concurrent connections" claim rather than assert it.
package main

import (
	"flag"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	addr := flag.String("addr", "localhost:9080", "host:port to connect to")
	count := flag.Int("n", 5000, "number of concurrent connections to open")
	hold := flag.Duration("hold", 500*time.Millisecond, "how long to hold each connection open")
	flag.Parse()

	var (
		wg        sync.WaitGroup
		succeeded atomic.Int64
		failed    atomic.Int64
	)

	start := time.Now()
	req := []byte("GET / HTTP/1.1\r\nHost: bench\r\nConnection: close\r\n\r\n")

	for i := 0; i < *count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			conn, err := net.DialTimeout("tcp", *addr, 5*time.Second)
			if err != nil {
				failed.Add(1)
				return
			}
			defer conn.Close()

			if _, err := conn.Write(req); err != nil {
				failed.Add(1)
				return
			}

			buf := make([]byte, 512)
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			if _, err := conn.Read(buf); err != nil {
				failed.Add(1)
				return
			}

			time.Sleep(*hold)
			succeeded.Add(1)
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	fmt.Printf("target:              %s\n", *addr)
	fmt.Printf("attempted conns:     %d\n", *count)
	fmt.Printf("succeeded:           %d\n", succeeded.Load())
	fmt.Printf("failed:              %d\n", failed.Load())
	fmt.Printf("wall time:           %s\n", elapsed.Round(time.Millisecond))
}
