package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
)

const capability = 10

const newLine = "\r\n"

var httpCodeMap = map[string]string{
	"200": "OK",
	"400": "Bad Request",
	"404": "Not Found",
	"500": "Internal Server Error",
	"501": "Not Implemented",
}

type Response struct {
	StatusCode string
	Headers    map[string]string
	BBody      []byte
}

func main() {
	start()
}

func start() {
	rawPort := "8081"
	if len(os.Args) > 1 {
		rawPort = os.Args[1]
	}
	if _, err := strconv.Atoi(rawPort); err != nil {
		rawPort = "8081"
	}

	listener, err := net.Listen("tcp", ":"+rawPort)
	if err != nil {
		panic(fmt.Sprintf("Error creating listner: %s", err))
	}

	defer listener.Close()
	var sem = make(chan int, capability)
	for {
		sem <- 1
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %s", err)
		}
		go proxyHTTPRequest(conn, sem)
	}
}
func proxyHTTPRequest(conn net.Conn, sem chan int) {
	defer func(conn net.Conn) {
		<-sem
		err := conn.Close()
		if err != nil {
			fmt.Printf("Error closing connection: %s", err)
		}
	}(conn)

	response := &Response{}
	response.Headers = make(map[string]string)
	response.StatusCode = "200"

	buffer := make([]byte, 4096)
	_, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading from connection")
		return
	}
	fmt.Println(string(buffer))

	// parse http request.
	request, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(buffer)))
	if err != nil {
		// 400 Bad Request
		response.Headers["Content-Type"] = "text/plain"
		response.StatusCode = "400"
		write(conn, response)
		return
	}
	// delete proxy header, we don't need to pass it to the real server.
	request.Header.Del("Proxy-Connection")

	if request.Method != "GET" {
		response.StatusCode = "501"
		write(conn, response)
		return
	}

	/**
	construct a new http request to real server, which means as a http client to send request.
	*/
	newRequest, err := http.NewRequest(request.Method, request.URL.String(), nil)
	if err != nil {
		response.StatusCode = "400"
		write(conn, response)
		return
	}
	// copy headers
	for k, v := range request.Header {
		newRequest.Header.Set(k, v[0])
	}

	resp, err := http.DefaultClient.Do(newRequest)
	if err != nil {
		fmt.Printf("Error proxying request to %s: %s", request.URL, err)
		response.StatusCode = "400"
		write(conn, response)
		return
	}
	defer resp.Body.Close()

	// copy response from real server and construct response.
	response.StatusCode = resp.Status
	// set headers
	for k, v := range resp.Header {
		response.Headers[k] = v[0]
	}
	// set body
	bodyBuffer, err := io.ReadAll(resp.Body)
	if err != nil {
		response.Headers["Content-Type"] = "text/plain"
		response.StatusCode = "400"
		return
	}
	response.BBody = bodyBuffer
	write(conn, response)
}

func write(conn net.Conn, response *Response) {
	statusCode := response.StatusCode
	conn.Write([]byte("HTTP/1.1 " + statusCode + " " + httpCodeMap[statusCode] + newLine))
	for k, v := range response.Headers {
		conn.Write([]byte(k + ": " + v + newLine))
	}
	conn.Write([]byte(newLine))
	if response.BBody != nil {
		conn.Write(response.BBody)
	}
}
