// Command-Line Interface - UI

package main

import (
	"flag"
	"fmt"
	"os"
)

type Config struct {
	IP   string // -a
	PORT int    // -p

	// joining chord ring
	JoinIP   string // --ja
	JoinPORT int    // --jp

	// stabilization timers (ms)
	StabilizeTime  int // --ts
	FixFingersTime int // --tff
	CheckPredTime  int // --tcp

	// successor list size
	SuccessorCount int // -r

	// optional ID override
	IDOveride string // -i
}

func ParseArgs() *Config {
	config := &Config{}

	flag.StringVar(&config.IP, "a", "", "IP-Address to bind to")
	flag.IntVar(&config.PORT, "p", 0, "Port to bind to")

	flag.StringVar(&config.JoinIP, "ja", "", "IP-Address of existing node to join")
	flag.IntVar(&config.JoinPORT, "jp", 0, "Port of existing node to join")

	flag.IntVar(&config.StabilizeTime, "ts", 0, "Time between Stabilize invocations (ms)")
	flag.IntVar(&config.FixFingersTime, "tff", 0, "Time between Fix Fingers invocations (ms)")
	flag.IntVar(&config.CheckPredTime, "tcp", 0, "Time between Predecessor invocations (ms)")

	flag.IntVar(&config.SuccessorCount, "r", 0, "Number of Successors to maintain")

	flag.StringVar(&config.IDOveride, "i", "", "Optional ID Override (40 hex char)")

	flag.Parse()

	// simple validate
	if config.IP == "" || config.PORT == 0 {
		fmt.Println("Usage: chord -a <address> -p <port> [--ja <join-address> --jp <join-port>]")
		os.Exit(1)
	}

	// if one join param is set, both must be set to for node to join existing ring
	if (config.JoinIP != "" && config.JoinPORT == 0) || (config.JoinIP == "" && config.JoinPORT != 0) {
		fmt.Println("Error: Both --ja and -jp must be specified together to join a ring")
		os.Exit(1)
	}

	// validate required parameters
	if config.StabilizeTime == 0 || config.FixFingersTime == 0 || config.CheckPredTime == 0 || config.SuccessorCount == 0 {
		fmt.Println("Error: Required Parameters: --ts, --tff, --tcp, -r")
		os.Exit(1)
	}

	// validate timer ranges (1 - 60 000 ms)
	if config.StabilizeTime < 1 || config.StabilizeTime > 60000 {
		fmt.Println("Error: --ts must be in range [1, 60 000]")
		os.Exit(1)
	}
	if config.FixFingersTime < 1 || config.FixFingersTime > 60000 {
		fmt.Println("Error: --tff must be in range [1, 60 000]")
		os.Exit(1)
	}
	if config.CheckPredTime < 1 || config.CheckPredTime > 60000 {
		fmt.Println("Error: --tcp must be in range [1, 60 000]")
		os.Exit(1)
	}

	// validate successor count range (1 - 32)
	if config.SuccessorCount < 1 || config.SuccessorCount > 32 {
		fmt.Println("Error: -r must be in range [1, 32]")
		os.Exit(1)
	}

	// validate ID override format (if provided, must be 40 hex chars)
	if config.IDOveride != "" {
		if len(config.IDOveride) != 40 {
			fmt.Println("Error: -i must be exactly 40 hex characters")
			os.Exit(1)
		}

		// check if all characters are hex [0-9a-fA-F]
		for _, c := range config.IDOveride {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				fmt.Println("Error: -i must contain only hex characters [0-9a-fA-F]")
				os.Exit(1)
			}
		}
	}

	return config
}
