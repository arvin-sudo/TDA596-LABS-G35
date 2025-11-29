package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
)

var filePath = "."

func main() {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "http server listen err: %s \n", err)
	}
	fileServerPort := listener.Addr().(*net.TCPAddr).Port
	fmt.Fprintf(os.Stdout, "%d", fileServerPort)
	http.Serve(listener, http.FileServer(http.Dir(filePath)))

	defer listener.Close()
}
