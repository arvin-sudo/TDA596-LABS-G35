// Chord-Client Main

package main

import (
	"fmt"
)

func main() {
	fmt.Println("Chord node starting..")

	config := ParseArgs()

	fmt.Printf("IP-Address: %s\n", config.IP)
	fmt.Printf("Port: %d\n", config.PORT)

	// create node
	node := NewNode(config.IP, config.PORT)
	node.PrintInfo()
}
