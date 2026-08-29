package main

import (
	"log"

	"github.com/LakshBharani/Load-Balancer/servers"
)

func main() {
	if err := servers.RunServers(5); err != nil {
		log.Fatal(err)
	}
}
