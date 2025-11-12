package main

import (
	"bufio"         // buffered reading
	"bytes"         // create reader from buffer
	"fmt"           // print to console
	"net"           // tcp listener
	"net/http"      // parse http request
	"os"            // OS system functions
	"path/filepath" // file path and extensions operations
	"slices"        // slice operations
	"strconv"       // string to int conversion
	"strings"       // string operations
)

const maxConcurrentConnections = 10
const newLine = "\r\n"

var httpCodeMap = map[string]string {
	"200": "OK",
	"400": "Bad Request",
	"404": "Not Found",
	"500": "Internal Server Error",
	"501": "Not Implemented",
}

type httpResponse struct {
	StatusCode string
	Headers map[string]string
	Body *string
	BinaryBody []byte
}

// validation func that checks if the file extension is allowed by the server
func isValidExtension(filename string) bool {
	extension := filepath.Ext(filename)
	// allowed extensions
	allowedExtensions := []string{".html", ".txt", ".gif", ".jpeg", ".jpg", ".css"}
	return slices.Contains(allowedExtensions, extension)
}

// getContentType returns the correct Content-Type header based on file extension
func getContentType(filename string) string {
	extension := filepath.Ext(filename)

	switch extension {
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
		panic("BUG: Invalid file extension reached getContentType")
	}
}

// setContentType sets the correct Content-Type header based on file extension
func setContentType(response *httpResponse, extension string) {
	switch extension {
	case ".html":
		response.Headers["Content-Type"] = "text/html"
	case ".txt":
		response.Headers["Content-Type"] = "text/plain"
	case ".gif":
		response.Headers["Content-Type"] = "image/gif"
	case ".jpeg", ".jpg":
		response.Headers["Content-Type"] = "image/jpeg"
	case ".css":
		response.Headers["Content-Type"] = "text/css"
	default:
		panic("BUG: Invalid file extension reached setContentType")
	}
}

// writeHttpResponse sends an HTTP response to the client connection
func writeHttpResponse(conn net.Conn, response *httpResponse) {
	statusCode := response.StatusCode

	// status line: HTTP/1.1 <status code> <status text>
	conn.Write([]byte("HTTP/1.1 " + statusCode + " " + httpCodeMap[statusCode] + newLine))
	
	// headers
	for header, value := range response.Headers {
		conn.Write([]byte(header + ": " + value + newLine))
	}

	// blank line to separate headers from body
	conn.Write([]byte(newLine))

	// body (if exists)
	if response.Body != nil {
		conn.Write([]byte(*response.Body))
	} else if response.BinaryBody != nil {
		conn.Write(response.BinaryBody)
	}
}

// strPtr create pointer to string, helper function for httpResponse.body
func strPtr(s string) *string {
	return &s
}

func handleClientConnection(conn net.Conn, semaphore chan struct{}) {
	// close connection and release semaphore slot when function exits
	defer func() {
		conn.Close()
		<-semaphore // release the semaphore slot
	}()

	// read raw request data
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading from client connection:", err)
		return
	}

	// parse HTTP request using net/http package
	request, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(buffer[:n])))
	if err != nil {
		// 400 Bad Request - malformed HTTP request
		resp := &httpResponse{
			StatusCode: "400",
			Headers:    map[string]string{"Content-Type": "text/plain"},
			Body:       strPtr("400 Bad Request: Malformed HTTP request\n"),
		}
		writeHttpResponse(conn, resp)
		return
	}

	// extract method and path from parsed request
	method := request.Method
	path := request.RequestURI

	fmt.Println("Parsed Method:", method)
	fmt.Println("Parsed Path:", path)

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

		// read body content from parsed request
		bodyBytes, err := bufio.NewReader(request.Body).ReadBytes('\x00')
		if err != nil && len(bodyBytes) == 0 {
			fmt.Println("Error: POST request has no body.")
			resp := &httpResponse{
				StatusCode: "400",
				Headers:    map[string]string{"Content-Type": "text/plain"},
				Body:       strPtr("400 Bad Request: No body found\n"),
			}
			writeHttpResponse(conn, resp)
			return
		}

		// trim null byte if present
		bodyContent := strings.TrimRight(string(bodyBytes), "\x00")

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
		err = os.WriteFile(filepath, []byte(bodyContent), 0644)
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
