package servers

import (
	"fmt"
	"log"
	"net/http"
	"sync"
)

const maxServers = 10

type ServerList struct {
	Ports []int
}

func (s *ServerList) Populate(amount int) error {
	if amount <= 0 {
		return fmt.Errorf("amount of servers must be positive, got %d", amount)
	}

	if amount > maxServers {
		return fmt.Errorf("amount of servers cannot be greater than %d, got %d", maxServers, amount)
	}

	for i := 0; i < amount; i++ {
		s.Ports = append(s.Ports, 8080+i)
	}
	return nil
}

func (s *ServerList) Pop() (int, bool) {
	if len(s.Ports) == 0 {
		return 0, false
	}
	port := s.Ports[0]
	s.Ports = s.Ports[1:]
	return port, true
}

func RunServers(amount int) error {
	// Server list Object
	var myServerList ServerList
	if err := myServerList.Populate(amount); err != nil {
		return err
	}

	var wg sync.WaitGroup

	// Pop on the main goroutine so the slice is never mutated concurrently.
	for {
		port, ok := myServerList.Pop()
		if !ok {
			break
		}

		wg.Add(1)
		go makeServers(port, &wg)
	}

	wg.Wait()
	return nil
}

func makeServers(port int, wg *sync.WaitGroup) {
	defer wg.Done()

	r := http.NewServeMux()
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Server %d", port)
	})
	server := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}

	log.Printf("server listening on :%d", port)
	if err := server.ListenAndServe(); err != nil {
		log.Printf("server on :%d stopped: %v", port, err)
	}
}
