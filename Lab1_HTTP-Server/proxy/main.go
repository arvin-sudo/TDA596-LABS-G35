package main

import (
	"bufio" // buffered reading
	"bytes" // create reader from buffer
	"fmt"   // print to console
	"io"       // read response body (kommer användas i Del B)
	"net"      // tcp listener
	"net/http" // parse http request
	"os"       // OS system functions
	"strconv"  // string to int conversion
)

const maxConcurrentConnections = 10
const newLine = "\r\n"

var httpCodeMap = map[string]string{
	"200": "OK",
	"400": "Bad Request",
	"404": "Not Found",
	"500": "Internal Server Error",
	"501": "Not Implemented",
	"502": "Bad Gateway",
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

// buildErrorResponse creates an error response with given status code and message
func buildErrorResponse(statusCode, message string) *httpResponse {
	return &httpResponse{
		StatusCode: statusCode,
		Headers:    map[string]string{"Content-Type": "text/plain"},
		Body:       strPtr(statusCode + " " + httpCodeMap[statusCode] + ": " + message + "\n"),
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
		fmt.Println("Usage: ./proxy <port>")
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

	fmt.Println("Proxy is listening on port", port)
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

// forwardRequestToOrigin sends the clients request to the origin server
// and returns the response to proxy
func forwardRequestToOrigin(method, url string) (*http.Response, error) {
	// create new request to forward to origin server
	originRequest, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	// send request to origin server and WAIT on get response (blocking call)
	originResponse, err := http.DefaultClient.Do(originRequest)
	if err != nil {
		return nil, err
	}

	// return origin server response to proxy
	return originResponse, nil
}

// buildProxyResponse creates an httpResponse from the origin server's response
func buildProxyResponse(originResponse *http.Response) (*httpResponse, error) {
	// read response body from origin server
	bodyBytes, err := io.ReadAll(originResponse.Body)
	if err != nil {
		return nil, err
	}

	// build proxy response with status and body to send back to client
	response := &httpResponse{
		StatusCode: originResponse.Status,
		Headers:    map[string]string{},
		BinaryBody: bodyBytes,
	}

	// copy headers from origin response to proxy response
	// copies Content-Type, Content-Length, Date, etc.
	for headerName, headerValues := range originResponse.Header {
		if len(headerValues) > 0 {
			response.Headers[headerName] = headerValues[0]
		}
	}

	// send to handleProxyConnection
	return response, nil
}

// handleProxyConnection handles a single client connection through the proxy
func handleProxyConnection(conn net.Conn, semaphore chan struct{}) {
	// close connection and release semaphore slot when function exits
	defer func() {
		conn.Close()
		<-semaphore // release the semaphore slot
	}()

	// read raw request data
	buffer := make([]byte, 4096)
	data, err := conn.Read(buffer)
	if err != nil {
		return
	}

	// parse HTTP request using net/http package
	request, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(buffer[:data])))
	if err != nil {
		writeHttpResponse(conn, buildErrorResponse("400", "Malformed HTTP request"))
		return
	}

	// extract method and URL from parsed request
	method := request.Method
	url := request.URL.String()

	// validate HTTP method - proxy only supports GET
	if method != "GET" {
		writeHttpResponse(conn, buildErrorResponse("501", "Only GET method is supported"))
		return
	}

	// forward request to origin server
	originResponse, err := forwardRequestToOrigin(method, url)
	if err != nil {
		writeHttpResponse(conn, buildErrorResponse("502", "Could not reach origin server"))
		return
	}
	defer originResponse.Body.Close()

	// build proxy response from origin response
	response, err := buildProxyResponse(originResponse)
	if err != nil {
		writeHttpResponse(conn, buildErrorResponse("502", "Error reading origin response"))
		return
	}

	// send response back to client
	writeHttpResponse(conn, response)
}

// runProxy accepts connections in an infinite loop and spawns goroutines to handle each client
func runProxy(listener net.Listener, semaphore chan struct{}) {
	// wait on client connections (multiple clients by for-loop)
	for {
		conn, err := listener.Accept()
		// handle error
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		// acquire semaphore slot (blocks if 10 goroutines are already running)
		semaphore <- struct{}{}

		// spawn goroutine to handle this proxy connection
		go handleProxyConnection(conn, semaphore)
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

	// RUN PROXY
	runProxy(listener, semaphore)
}
