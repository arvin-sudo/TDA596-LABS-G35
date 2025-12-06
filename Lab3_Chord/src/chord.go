// Chord-Client Main

package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Chord node starting..n")

	config := ParseArgs()

	// create node
	node := NewNode(config.IP, config.PORT)
	node.PrintInfo()

	// start RPC server
	err := node.StartRPCServer()
	if err != nil {
		fmt.Printf("Error starting RPC-Server: %v\n", err)
		return
	}

	// create new or join chord ring based on config
	if config.JoinIP == "" {
		// no join address = create new ring
		node.Create()
	} else {
		// join existing ring
		joinRing := fmt.Sprintf("%s:%d", config.JoinIP, config.JoinPORT)
		err := node.Join(joinRing)
		if err != nil {
			fmt.Printf("Error joining ring: %v\n", err)
			return
		}
	}

	node.PrintInfo()

	// start goroutine stabilization in background
	go func() {
		// create timer that ticks every 3 seconds
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for {
			// wait on next tick
			<-ticker.C
			// run stabilize when ticks
			node.Stabilize()
		}
	}()

	// keep running or program exits
	fmt.Println("Node running. Ctrl+C to stop")
	select {} // block forever
}
