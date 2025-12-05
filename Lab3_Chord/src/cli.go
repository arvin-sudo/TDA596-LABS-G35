// Command-Line Interface - UI

package main

import (
	"flag"
	"fmt"
	"os"
)

type Config struct {
	IP       string //-a
	PORT     int    // -p
	JoinIP   string // --ja
	JoinPORT int    // --jp
}

func ParseArgs() *Config {
	config := &Config{}

	flag.StringVar(&config.IP, "a", "", "IP-Address to bind to")
	flag.IntVar(&config.PORT, "p", 0, "Port to bind to")
	flag.StringVar(&config.JoinIP, "ja", "", "IP-Address of existing node to join")
	flag.IntVar(&config.JoinPORT, "jp", 0, "Port of existing node to join")

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

	return config
}
