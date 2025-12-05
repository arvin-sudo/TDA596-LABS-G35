// Command-Line Interface - UI

package main

import (
	"flag"
	"fmt"
	"os"
)

type Config struct {
	IP string
	PORT    int
}

func ParseArgs() *Config {
	config := &Config{}

	flag.StringVar(&config.IP, "a", "", "IP address to bind to")
	flag.IntVar(&config.PORT, "p", 0, "Port to bind to")

	flag.Parse()

	// simple validate
	if config.IP == "" || config.PORT == 0 {
		fmt.Println("Usage: chord -a <address> -p <ports>")
		os.Exit(1)
	}

	return config
}
