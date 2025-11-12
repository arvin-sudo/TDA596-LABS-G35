package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
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
	rawPort := "8082"
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
	var sem = make(chan int, capability)
	for {
		sem <- 1
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %s", err)
		}
		//fmt.Println("Accepted connection: one time")
		go handleTCPConnection(conn, sem)
	}
}

func handleTCPConnection(conn net.Conn, sem chan int) {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Printf("Error closing connection: %s", err)
		}
		<-sem
	}(conn)
	fmt.Printf("New connection from: %s\n", conn.RemoteAddr())
	fmt.Println("Accepted connection: one time")
	response := &Response{}
	response.Headers = make(map[string]string)
	response.StatusCode = "200"

	// make bigger for uploading 200kb image.
	buffer := make([]byte, 4096*4*10*1000)
	// trap: 1. postman reused tcp connection, so one clicked will trigger into two go routines.
	// 2. don't use for loop to receive data, the client will not close connection, conn.Read() will get block waiting a close request. then deadlock.
	n, err := conn.Read(buffer)
	fmt.Printf("n: %d err: %v\n", n, err)
	//if n > 0 {
	//buffer = append(buffer, tmp[:n]...)
	//}
	if err != nil {
		if errors.Is(err, io.EOF) {
			//break
		} else {
			fmt.Printf("Error reading from connection: %s", err)
			response.StatusCode = "500"
			return
		}
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
	if request.Method != "GET" && request.Method != "POST" {
		// 501 Not Implemented
		response.Headers["Content-Type"] = "text/plain"
		response.StatusCode = "501"
		write(conn, response)
		return
	}

	// post, get dispatch.
	if request.Method == "POST" {
		handlePostMethod(request, response, conn)
	} else if request.Method == "GET" {
		handleGetMethod(conn, uri, response)
	}

	// json. json.NewDecoder(request.Body).Decode()
	// maybe for advanced part as a proxy. http.NewRequestWithContext(context.Background(), "GET", "http://"+conn.LocalAddr().String(), nil)

	return
}

func handleGetMethod(conn net.Conn, uri string, response *Response) {
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
}
func handlePostMethod(request *http.Request, response *Response, conn net.Conn) {
	err := request.ParseMultipartForm(32 << 20)
	if err != nil {
		fmt.Printf("Error parsing multipart form: %s", err)
	}
	fileMap := request.MultipartForm.File
	for k, v := range fileMap {
		fmt.Println(k, v[0].Filename)
		multipartFile := v[0]
		filename := multipartFile.Filename
		sourceFD, err := multipartFile.Open()
		if err != nil {
			response.StatusCode = "500"
			write(conn, response)
			return
		}
		defer sourceFD.Close()

		targetFD, err := os.Create("files/" + filename)
		if err != nil {
			response.StatusCode = "500"
			write(conn, response)
			return
		}
		defer targetFD.Close()
		_, err = io.Copy(targetFD, sourceFD)
		if err != nil {
			response.StatusCode = "500"
			write(conn, response)
			return
		}

	}
	write(conn, response)
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
