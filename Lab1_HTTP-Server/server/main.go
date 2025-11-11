package main

import (
	//"bufio"		// buffered reading
	//"bytes"		// create reader from buffer
	"fmt" // print to console
	"net" // tcp listener

	//"net/http"	// parse http request
	"os"      // OS system functions
	"strings" // string operations
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

		// convert buffer to string
		requestStr := string(buffer[:data])

		// console test print
		fmt.Println("======== RAW REQUEST ========")
		fmt.Println(requestStr)
		fmt.Println("======== END REQUEST ========")

		// split the request string into lines
		lines := strings.Split(requestStr, "\r\n")

		// first line
		requestLine := lines[0] // e.g., "GET / HTTP/1.1"
		fmt.Println("Request Line:", requestLine)

		// split the request line into parts
		parts := strings.Fields(requestLine)

		var method string
		var path string


		if len(parts) >= 2 {
			method = parts[0] // e.g., "GET"
			path = parts[1]   // e.g., "/test.html"

			fmt.Println("Parsed Method:", method)
			fmt.Println("Parsed Path:", path)
		}

		// print the received data
		fmt.Println("Received data from client:")
		fmt.Println(string(buffer[:data]))

		fmt.Println("Client connected:")

		// // server-reply(write) to the client
		// reply := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nHello from server!\n"
		// conn.Write([]byte(reply))
		// fmt.Println("Response sent to client.")

		// parsed path to read file and reply with content to client
		filepath := "files" + path
		fmt.Println("Trying to read file:", filepath)

		fileContent, err := os.ReadFile(filepath)
		// handle error
		if err != nil {
			fmt.Println("File not found", filepath)
			response := "HTTP/1.1 404 Not Found\r\nContent-Type: text/plain\r\n\r\nFile Not Found\n"
			conn.Write([]byte(response))
		} else {
			// send file content as response
			fmt.Println("file found, sending", len(fileContent), "bytes")
			response := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\n" + string(fileContent)
			conn.Write([]byte(response))
		}

		fmt.Println("Response sent to client.")

		// close the connection when main exits
		conn.Close()
	}
}
