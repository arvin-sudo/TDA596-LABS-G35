package main

import (
	//"bufio"		// buffered reading
	//"bytes"		// create reader from buffer
	"fmt"     // print to console
	"net"     // tcp listener
	"strconv" // string to int conversion

	//"net/http"	// parse http request
	"os"            // OS system functions
	"path/filepath" // file path and extensions operations
	"strings"       // string operations
)

// validation func that checks if the file extension is allowed by the server
func isValidExtension(filename string) bool {
	ext := filepath.Ext(filename)

	// allowed extensions
	allowedExtensions := []string{".html", ".txt", ".gif", ".jpeg", ".jpg", ".css"}

	for _, allowed := range allowedExtensions {
		if ext == allowed {
			return true
		}
	}
	return false
}

// getContentType returns the correct Content-Type header based on file extension
func getContentType(filename string) string {
	ext := filepath.Ext(filename)

	switch ext {
	case ".html":
		return "text/html"
	case ".txt":
		return "text/plain"
	case ".gif":
		return "image/gif"
	case ".jpeg", ".jpg":
		return "image/jpeg"
	case ".css":
		return "text/css"
	default:
		return "application/octet-stream" // default binary type
	}
}

func handleClientConnection(conn net.Conn, semaphore chan struct{}) {
	defer func() {
		<-semaphore // release the semaphore slot
		conn.Close()
	}()

	// server-read the data from the client connection
	buffer := make([]byte, 1024)
	data, err := conn.Read(buffer)
	// handle error
	if err != nil {
		fmt.Println("Error reading from client connection:", err)
		return	
	}

	// convert buffer to string to be able to parse
	requestStr := string(buffer[:data])

	// console test print
	fmt.Println("======== RAW REQUEST ========")
	fmt.Println(requestStr)
	fmt.Println("======== END REQUEST ========")

	// PARSE
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
	// END PARSE

	// validate HTTP method GET and POST
	if method != "GET" && method != "POST" {
		fmt.Println("Unsupported HTTP method:", method)
		response := "HTTP/1.1 501 Not implemented\r\nContent-Type: text/plain\r\n\r\n501 Not Implemented: Method not supported\n"
		conn.Write([]byte(response))
		return
	}

	// handle POST method requests - upload and save files
	if method == "POST" {
		fmt.Println("Handling POST request for file upload.")

		// find where body starts
		bodyStartIndex := strings.Index(requestStr, "\r\n\r\n")
		// handle error
		if bodyStartIndex == -1 {
			fmt.Println("Error: Malformed POST request: no body found.")
			response := "HTTP/1.1 400 Bad Request\r\nContent-Type: text/plain\r\n\r\n400 Bad Request: No body found\n"
			conn.Write([]byte(response))
			return
		}

		// extract body content
		bodyStartIndex += 4 // skip the "\r\n\r\n"
		bodyContent := requestStr[bodyStartIndex:]

		// print body content to console for testing
		fmt.Println("POST body content:")
		fmt.Println(bodyContent)

		// build the file path
		filepath := "files" + path
		fmt.Println("Saving uploaded file to:", filepath)

		// validate file extension before saving
		if !isValidExtension(filepath) {
			fmt.Println("Invalid file extension for", filepath)
			response := "HTTP/1.1 400 Bad Request\r\nContent-Type: text/plain\r\n\r\n400 Bad Request: Invalid file extension\n"
			conn.Write([]byte(response))
			return
		}

		// write body content to file
		err := os.WriteFile(filepath, []byte(bodyContent), 0644)
		// handle error
		if err != nil {
			fmt.Println("Error writing file:", err)
			response := "HTTP/1.1 500 Internal Server Error\r\nContent-Type: text/plain\r\n\r\n500 Internal Server Error: Could not save file\n"
			conn.Write([]byte(response))
			return
		}

		// send success response to client
		fmt.Println("File uploaded successfully:", filepath)
		response := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nFile uploaded successfully\n"
		conn.Write([]byte(response))
		return	
	} else if method == "GET" {
		// parsed path to read file and reply with content to client
		filepath := "files" + path
		fmt.Println("Trying to read file:", filepath)

		// validate file extension before reading
		if !isValidExtension(filepath) {
			fmt.Println("Invalid file extension for", filepath)
			contentType := getContentType(filepath)
			response := "HTTP/1.1 400 Bad Request\r\nContent-Type: " + contentType + "\r\n\r\n400 Bad Request:\n"
			conn.Write([]byte(response))
			return
		}

		fileContent, err := os.ReadFile(filepath)
		// handle error
		if err != nil {
			fmt.Println("File not found", filepath)
			contentType := getContentType(filepath)
			response := "HTTP/1.1 404 Not Found\r\nContent-Type: " + contentType + "\r\n\r\nFile Not Found\n"
			conn.Write([]byte(response))
		} else {
			// send file content as response
			fmt.Println("File found, sending", len(fileContent), "bytes")
			contentType := getContentType(filepath)
			response := "HTTP/1.1 200 OK\r\nContent-Type: " + contentType + "\r\n\r\n" + string(fileContent)
			conn.Write([]byte(response))
		}

		fmt.Println("Response sent to client.")
	}
}


func main() {
	// PORT SETUP AND VALIDATION
	// check and get port from command line arguments
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./http-server <port>")
		os.Exit(1)
	}

	// set port from command line argument
	port := os.Args[1]

	// validate port number
	if _, err := strconv.Atoi(port); err != nil {
		fmt.Println("Invalid port number:", port)
		os.Exit(1)
	}

	// create a TCP listener on port
	listener, err := net.Listen("tcp", ":"+port)
	// handle error
	if err != nil {
		fmt.Println("Error starting TCP listener:", err)
		os.Exit(1)
	}

	// remember to close the listener when main exits
	defer listener.Close()
	
	fmt.Println("Server is listening on port", port)

	// CONCURRENCY SETUP
	// semaphore pattern to limit concurrency to 10 simultaneous connections
	const maxConcurrentConnections = 10
	semaphore := make(chan struct{}, maxConcurrentConnections) // buffered channel acting as semaphore

	// wait on client connections (multiple clients by for-loop)
	for {
		conn, err := listener.Accept()
		// handle error
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		// acquire semaphore slot (blocks if 10 goroutines are already running)
		// struct{} is zero-sized, so this is memory efficient
		// struct{}{}, first is type declaration (empty field), second is value initialization
		semaphore <- struct{}{}

		// spawn goroutine to handle this client connection
		go handleClientConnection(conn, semaphore)
	}
}
