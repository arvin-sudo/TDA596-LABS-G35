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
	Body       *string
	BBody      []byte
}

func main() {
	rawPort := "8081"
	if len(os.Args) > 1 {
		rawPort = os.Args[1]
	}
	if _, err := strconv.Atoi(rawPort); err != nil {
		panic(fmt.Sprintf("Invalid port number: %s", rawPort))
	}

	listener, err := net.Listen("tcp", ":"+rawPort)
	if err != nil {
		panic(fmt.Sprintf("Error creating listner: %s", err))
	}

	defer listener.Close()
	cur := 0
	for cur < capability {
		done := make(chan bool)
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %s", err)
		}
		cur++
		go proxyHTTPRequest(conn, done)
		<-done
		cur--
	}
}

func proxyHTTPRequest(conn net.Conn, done chan bool) {
	defer func(conn net.Conn) {
		done <- true
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

	//
	request, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(buffer)))
	if err != nil {
		// 400 Bad Request
		response.Headers["Content-Type"] = "text/plain"
		response.StatusCode = "400"
		write(conn, response)
		return
	}
	request.Header.Del("Proxy-Connection")
	fmt.Println(request.RequestURI)
	newRequest, err := http.NewRequest(request.Method, request.URL.String(), nil)
	if err != nil {
		response.StatusCode = "400"
		write(conn, response)
		return
	}
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
	response.StatusCode = resp.Status
	// set headers
	for k, v := range resp.Header {
		response.Headers[k] = v[0]
		//fmt.Println(k, v)
	}
	// set body
	bodyBuffer, err := io.ReadAll(resp.Body)
	fmt.Println(string(bodyBuffer))
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
	if response.Body != nil {
		conn.Write([]byte(*response.Body))
	}
	if response.BBody != nil {
		conn.Write(response.BBody)
	}

}
