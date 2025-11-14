package main

import (
	"bufio"         // buffered reading
	"bytes"         // create reader from buffer
	"fmt"           // print to console
	"io"            // read request body
	"net"           // tcp listener
	"net/http"      // parse http request
	"os"            // OS system functions
	"path/filepath" // file path and extensions operations
	"slices"        // slice operations
	"strconv"       // string to int conversion
)

const maxConcurrentConnections = 10
const newLine = "\r\n"

var httpCodeMap = map[string]string{
	"200": "OK",
	"400": "Bad Request",
	"404": "Not Found",
	"500": "Internal Server Error",
	"501": "Not Implemented",
}

type httpResponse struct {
	StatusCode string
	Headers    map[string]string
	Body       *string
	BinaryBody []byte
}

// ================
// HELPER FUNCTIONS
// ================

// strPtr create pointer to string, helper function for httpResponse.body
func strPtr(s string) *string {
	return &s
}

// isValidExtension checks if the file extension is allowed by the server
func isValidExtension(filename string) bool {
	extension := filepath.Ext(filename)
	// allowed extensions
	allowedExtensions := []string{".html", ".txt", ".gif", ".jpeg", ".jpg", ".css"}
	return slices.Contains(allowedExtensions, extension)
}

// setContentType sets the correct Content-Type header based on file extension
func setContentType(response *httpResponse, extension string) {
	switch extension {
	case ".html":
		response.Headers["Content-Type"] = "text/html; charset=UTF-8"
	case ".txt":
		response.Headers["Content-Type"] = "text/plain; charset=UTF-8"
	case ".gif":
		response.Headers["Content-Type"] = "image/gif"
	case ".jpeg", ".jpg":
		response.Headers["Content-Type"] = "image/jpeg"
	case ".css":
		response.Headers["Content-Type"] = "text/css; charset=UTF-8"
	default:
		panic("BUG: Invalid file extension reached setContentType")
	}
}

// buildErrorResponse creates an error response with given status code and message
func buildErrorResponse(statusCode, message string) *httpResponse {
	return &httpResponse{
		StatusCode: statusCode,
		Headers:    map[string]string{"Content-Type": "text/plain"},
		Body:       strPtr(statusCode + " " + httpCodeMap[statusCode] + ": " + message + "\n"),
	}
}

// buildTextResponse creates a success response with plain text body
func buildTextResponse(message string) *httpResponse {
	return &httpResponse{
		StatusCode: "200",
		Headers:    map[string]string{"Content-Type": "text/plain"},
		Body:       strPtr(message),
	}
}

// buildFileResponse creates a response for serving files with correct Content-Type
func buildFileResponse(fileContent []byte, extension string) *httpResponse {
	resp := &httpResponse{
		StatusCode: "200",
		Headers:    map[string]string{},
		BinaryBody: fileContent,
	}
	setContentType(resp, extension)
	return resp
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

	// body or binaryBody (if exists)
	if response.Body != nil {
		conn.Write([]byte(*response.Body))
	} else if response.BinaryBody != nil {
		conn.Write(response.BinaryBody)
	}
}

// ============================================================================
// SETUP FUNCTIONS
// ============================================================================

// setupPort validates and returns the port from command-line arguments
func setupPort() string {
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
	return port
}

// setupTCPListener creates and returns a TCP listener on the given port
func setupTCPListener(port string) net.Listener {
	// create a TCP listener on port
	listener, err := net.Listen("tcp", ":"+port)
	// handle error
	if err != nil {
		fmt.Println("Error starting TCP listener:", err)
		os.Exit(1)
	}

	fmt.Println("Server is listening on port", port)
	return listener
}

// setupConcurrency creates and returns a semaphore channel for limiting concurrent connections
func setupConcurrency() chan struct{} {
	// semaphore pattern to limit concurrency to 10 simultaneous connections
	semaphore := make(chan struct{}, maxConcurrentConnections) // buffered channel acting as semaphore
	return semaphore
}

// ============================================================================
// CONNECTION & REQUEST HANDLING
// ============================================================================

// handleGETRequest handles GET requests for file retrieval
func handleGETRequest(path string) *httpResponse {
	filePath := "files" + path
	fmt.Println("Trying to read file:", filePath)

	// validate file extension before reading
	if !isValidExtension(filePath) {
		fmt.Println("Invalid file extension for", filePath)
		return buildErrorResponse("400", "Invalid file extension")
	}

	// read file from disk
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println("File not found:", filePath)
		return buildErrorResponse("404", "File does not exist")
	}

	// success - return file with correct content type
	fmt.Println("File found, sending", len(fileContent), "bytes")
	extension := filepath.Ext(path)
	return buildFileResponse(fileContent, extension)
}

// handlePOSTRequest handles POST requests for file upload
func handlePOSTRequest(request *http.Request, path string) *httpResponse {
	fmt.Println("Handling POST request for file upload.")

	// read body content from request
	bodyBytes, err := io.ReadAll(request.Body)
	if err != nil || len(bodyBytes) == 0 {
		fmt.Println("Error: POST request has no body.")
		return buildErrorResponse("400", "No body found")
	}

	bodyContent := string(bodyBytes)
	fmt.Println("POST body content:")
	fmt.Println(bodyContent)

	// build file path and validate extension
	filePath := "files" + path
	fmt.Println("Saving uploaded file to:", filePath)

	if !isValidExtension(filePath) {
		fmt.Println("Invalid file extension for", filePath)
		return buildErrorResponse("400", "Invalid file extension")
	}

	// write file to disk
	err = os.WriteFile(filePath, []byte(bodyContent), 0644)
	if err != nil {
		fmt.Println("Error writing file:", err)
		return buildErrorResponse("500", "Could not save file")
	}

	// success
	fmt.Println("File uploaded successfully:", filePath)
	return buildTextResponse("File uploaded successfully\n")
}

// handleClientConnection handles a single client connection
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
		writeHttpResponse(conn, buildErrorResponse("400", "Malformed HTTP request"))
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
		writeHttpResponse(conn, buildErrorResponse("501", "Method not supported"))
		return
	}

	// route to appropriate handler based on method
	switch method {
	case "POST":
		writeHttpResponse(conn, handlePOSTRequest(request, path))
	case "GET":
		writeHttpResponse(conn, handleGETRequest(path))
	}
}

// runServer accepts connections in an infinite loop and spawns goroutines to handle each client
func runServer(listener net.Listener, semaphore chan struct{}) {
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

func main() {
	// PORT SETUP AND VALIDATION
	port := setupPort()

	// TCP LISTENER SETUP
	listener := setupTCPListener(port)
	defer listener.Close()

	// CONCURRENCY SETUP
	semaphore := setupConcurrency()

	// RUN SERVER
	runServer(listener, semaphore)
}
