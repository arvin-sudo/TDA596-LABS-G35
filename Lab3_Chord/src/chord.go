// Chord-Client Main

package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Chord Node Starting..")

	config := ParseArgs()

	// create node
	node := NewNode(config.IP, config.PORT)

	// start RPC server
	err := node.StartRPCServer()
	if err != nil {
		fmt.Printf("Error starting RPC-Server: %v\n", err)
		return
	}

	// Note: StartRPCServer returns after net.Listen() succeeds,
	// so the server is already listening at this point.
	// Small delay to ensure http.Serve goroutine is fully started.
	time.Sleep(100 * time.Millisecond)

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

	node.PrintState()

	// start goroutine stabilization in background
	go func() {
		// create 3 separate timers
		// one timer for every func
		stabilizeTicker := time.NewTicker(time.Duration(config.StabilizeTime) * time.Millisecond)
		fixFingersTicker := time.NewTicker(time.Duration(config.FixFingersTime) * time.Millisecond)
		checkPredTicker := time.NewTicker(time.Duration(config.CheckPredTime) * time.Millisecond)

		defer stabilizeTicker.Stop()
		defer fixFingersTicker.Stop()
		defer checkPredTicker.Stop()

		for {
			select {
			case <-stabilizeTicker.C:
				node.Stabilize()

			case <-fixFingersTicker.C:
				// todo: call fixFingers

			case <-checkPredTicker.C:
				// todo: call checkPred
			}
		}
	}()

	// command loop - read commands from user
	fmt.Println("--- Node Running... ---")
	node.CommandLoop()
}
