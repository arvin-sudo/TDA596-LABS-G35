package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	// TODO: implement port from command line arguments later
	port := "8080"

	// create a TCP listener on port 8080
	listener, err := net.Listen("tcp", ":"+port)
	// handle error
	if err != nil {
		fmt.Println("Error starting TCP listener:", err)
		os.Exit(1)
	}

	// remember to close the listener when main exits
	defer listener.Close()

	fmt.Println("Server is listening on port", port)

	// wait on client connections (multiple clients by for-loop)
	for {
		conn, err := listener.Accept()
		// handle error
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		// TODO: move all handling here(read, write, close)

		// server-read the data from the client connection
		buffer := make([]byte, 1024)
		data, err := conn.Read(buffer)
		// handle error
		if err != nil {
			fmt.Println("Error reading from client connection:", err)
			conn.Close()
			continue
		}

		// print the received data
		fmt.Println("Received data from client:")
		fmt.Println(string(buffer[:data]))

		fmt.Println("Client connected:")

		// server-reply(write) to the client
		reply := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nHello from server!\n"
		conn.Write([]byte(reply))
		fmt.Println("Reply sent to client.")

		// close the connection when main exits
		conn.Close()
	}
}
