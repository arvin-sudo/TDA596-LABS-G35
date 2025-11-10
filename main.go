package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	rawPort := "8080"
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
		go handleTCPConnection(conn, done)
		<-done
		cur--
	}
}

func handleTCPConnection(conn net.Conn, done chan<- bool) {
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
		fmt.Printf("Error reading from connection: %s", err)
		response.Headers["Content-Type"] = "text/plain"
		response.StatusCode = "400"
		write(conn, response)
		return
	}
	fmt.Println(string(buffer))

	// parse to http protocol

	request, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(buffer)))
	if err != nil {
		// 400 Bad Request
		response.Headers["Content-Type"] = "text/plain"
		response.StatusCode = "400"
		write(conn, response)
		return
	}

	// check extension
	uri := request.RequestURI
	fmt.Println(uri)
	if !strings.HasSuffix(uri, ".html") && !strings.HasSuffix(uri, ".txt") && !strings.HasSuffix(uri, ".gif") &&
		!strings.HasSuffix(uri, ".jpeg") && !strings.HasSuffix(uri, ".jpg") && !strings.HasSuffix(uri, ".css") {
		// 400 Bad Request
		conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n"))
		conn.Write([]byte("Content-Type: text/plain; charset=utf-8\r\n"))
		return
	}

	// check method
	fmt.Println(request.Method)
	if request.Method != "GET" && request.Method != "POST" {
		// 501 Not Implemented
		response.Headers["Content-Type"] = "text/plain"
		response.StatusCode = "501"
		write(conn, response)
		return
	}

	// find file
	fileData, err := os.ReadFile(uri[1:])
	if err != nil {
		// 404 not found
		response.Headers["Content-Type"] = "text/plain"
		response.StatusCode = "404"
		write(conn, response)
		return
	}

	// construct response
	// set body
	response.BBody = fileData

	// set headers. (only for content-type right now)
	ext := filepath.Ext(uri)
	fmt.Println(ext)
	setContentType(response, ext)

	write(conn, response)

	// json. json.NewDecoder(request.Body).Decode()
	// maybe for advanced part as a proxy. http.NewRequestWithContext(context.Background(), "GET", "http://"+conn.LocalAddr().String(), nil)

	return
}

func setContentType(response *Response, ext string) {
	if ext == ".html" {
		response.Headers["Content-Type"] = "text/html; charset=utf-8"
	} else if ext == ".txt" {
		response.Headers["Content-Type"] = "text/plain; charset=utf-8"
	} else if ext == ".gif" {
		response.Headers["Content-Type"] = "image/gif"
	} else if ext == ".jpeg" || ext == ".jpg" {
		response.Headers["Content-Type"] = "image/jpeg"
	} else if ext == ".css" {
		response.Headers["Content-Type"] = "text/css"
	}

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
