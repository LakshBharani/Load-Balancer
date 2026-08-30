// Command loadtest fires N concurrent HTTP requests at a target for a
// fixed duration, tracking per-request latency, and reports throughput
// and latency percentiles.
package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	target := flag.String("target", "http://localhost:9080", "URL to hit")
	concurrency := flag.Int("c", 200, "concurrent workers")
	duration := flag.Duration("d", 10*time.Second, "test duration")
	flag.Parse()

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        *concurrency * 2,
			MaxIdleConnsPerHost: *concurrency * 2,
			DisableKeepAlives:   false,
		},
	}

	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		latencies []time.Duration
		successes atomic.Int64
		failures  atomic.Int64
	)

	stop := time.Now().Add(*duration)
	start := time.Now()

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(stop) {
				reqStart := time.Now()
				resp, err := client.Get(*target)
				elapsed := time.Since(reqStart)

				if err != nil {
					failures.Add(1)
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					failures.Add(1)
					continue
				}

				successes.Add(1)
				mu.Lock()
				latencies = append(latencies, elapsed)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	totalElapsed := time.Since(start)

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	pct := func(p float64) time.Duration {
		if len(latencies) == 0 {
			return 0
		}
		idx := int(float64(len(latencies)) * p)
		if idx >= len(latencies) {
			idx = len(latencies) - 1
		}
		return latencies[idx]
	}

	total := successes.Load() + failures.Load()
	fmt.Printf("target:        %s\n", *target)
	fmt.Printf("concurrency:   %d\n", *concurrency)
	fmt.Printf("duration:      %s\n", totalElapsed.Round(time.Millisecond))
	fmt.Printf("total reqs:    %d\n", total)
	fmt.Printf("successes:     %d\n", successes.Load())
	fmt.Printf("failures:      %d\n", failures.Load())
	fmt.Printf("throughput:    %.1f req/s\n", float64(total)/totalElapsed.Seconds())
	fmt.Printf("p50 latency:   %s\n", pct(0.50))
	fmt.Printf("p90 latency:   %s\n", pct(0.90))
	fmt.Printf("p99 latency:   %s\n", pct(0.99))
	fmt.Printf("max latency:   %s\n", latencies[len(latencies)-1])
}
