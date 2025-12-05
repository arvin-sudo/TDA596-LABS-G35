// Chord-Client Main

package main

import (
	"fmt"
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

	// keep running or program exits
	fmt.Println("Node running. Ctrl+C to stop")
	select {} // block forever
}
