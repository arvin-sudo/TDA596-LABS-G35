// Chord-Client Main

package main

import (
	"fmt"
)

func main() {
	fmt.Println("Chord node starting..")

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

	// keep running or program exits
	fmt.Println("Node running. Ctrl+C to stop")
	select {} // block forever
}
