package main

import (
	"lb/control"
	"lb/misc"
	"log"
	"sync"
)

// The main entry of this program
func main() {
	// Print our load balancer's logo
	misc.PrintLBLogo()

	// Create a sync.WaitGroup for determining all goroutine's termiantion
	var wg sync.WaitGroup
	wg.Add(1)

	// Start up our control server
	controller := control.New()
	err := controller.Run(&wg)
	if err != nil {
		log.Fatalf("Could not run control server: %v", err)
		return
	}
}
