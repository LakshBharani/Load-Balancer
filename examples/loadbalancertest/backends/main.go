// Command backends starts a handful of plain HTTP servers that identify
// themselves by id — dummy targets for exercising the load balancer
// locally without Docker. Ports and ids match examples/loadbalancertest/config.yaml.
package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
)

type server struct {
	id   string
	port int
}

var servers = []server{
	{"srv-1", 8081},
	{"srv-2", 8082},
	{"srv-3", 8083},
	{"srv-4", 8084},
	{"srv-5", 8085},
	{"srv-6", 8086},
	{"srv-7", 8087},
	{"srv-8", 8088},
}

func main() {
	var wg sync.WaitGroup
	wg.Add(len(servers))

	for _, s := range servers {
		go run(s, &wg)
	}

	wg.Wait()
}

func run(s server, wg *sync.WaitGroup) {
	defer wg.Done()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s\n", s.id)
	})

	addr := fmt.Sprintf(":%d", s.port)
	httpServer := &http.Server{Addr: addr, Handler: mux}

	log.Printf("%s listening on %s", s.id, addr)
	if err := httpServer.ListenAndServe(); err != nil {
		log.Printf("%s stopped: %v", s.id, err)
	}
}
