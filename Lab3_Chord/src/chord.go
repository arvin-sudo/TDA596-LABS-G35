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

	// Test hash func
	address := fmt.Sprintf("%s:%d", config.IP, config.PORT)
	id := Hash(address)
	fmt.Printf("Node ID: %s\n", IDToString(id))
}
