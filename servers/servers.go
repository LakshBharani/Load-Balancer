package servers

import (
	"fmt"
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
	myServerList.Populate(amount)

	var wg sync.WaitGroup
	wg.Add(amount)
	defer wg.Wait()

	for i := 0; i < amount; i++ {
		go makeServers(&myServerList, &wg)
	}

	return nil
}

func makeServers(sl *ServerList, wg *sync.WaitGroup) {
	r := http.NewServeMux()
	defer wg.Done()

	port, _ := sl.Pop()
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Server %d", port)
	})
	server := http.Server{
		Addr:    fmt.Sprintf(":808%d", port),
		Handler: r,
	}

	server.ListenAndServe()
}
