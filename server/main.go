package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
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
			continue
		}
		//fmt.Println("Accepted connection: one time")
		go handleTCPConnection(conn, sem, rand.IntN(100))
	}
}

func handleTCPConnection(conn net.Conn, sem chan int, requestId int) {
	defer func(conn net.Conn) {
		if r := recover(); r != nil {
			fmt.Printf("goroutine panic: %v\n%s", r, debug.Stack())
		}
		err := conn.Close()
		if err != nil {
			fmt.Printf("Error closing connection: %s", err)
		}
		<-sem
	}(conn)
	fmt.Printf("[%d] New connection from: %s\n", requestId, conn.RemoteAddr())
	fmt.Printf("[%d] Accepted connection: one time\n", requestId)
	response := &Response{}
	response.Headers = make(map[string]string)
	response.StatusCode = "200"

	// 20M. make bigger for uploading 200kb image.
	buffer := make([]byte, 20*1024*1024)
	// trap: 1. postman reused tcp connection, so one clicked will trigger into two go routines.
	// 2. don't use for loop to receive data, the client will not close connection, conn.Read() will get block waiting a close request. then deadlock.
	n, err := conn.Read(buffer)
	fmt.Printf("[%d] n: %d err: %v\n", requestId, n, err)
	//if n > 0 {
	//buffer = append(buffer, tmp[:n]...)
	//}
	if err != nil {
		fmt.Printf("[%d] Error reading from connection: %s", requestId, err)
		response.StatusCode = "500"
		write(conn, response)
		return
	}

	fmt.Printf("[%d] buffer is readed.\n", requestId)
	// fmt.Println(string(buffer))

	// parse to http protocol

	request, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(buffer)))
	buffer = nil
	if err != nil {
		// 400 Bad Request
		fmt.Printf("[%d] Error parsing request: %s", requestId, err)
		response.Headers["Content-Type"] = "text/plain"
		response.StatusCode = "400"
		write(conn, response)
		return
	}

	fmt.Printf("[%d] construct request ok.\n", requestId)

	// check extension
	uri := request.RequestURI
	fmt.Printf("[%d] uri: %s\n", requestId, uri)
	if !strings.HasSuffix(uri, ".html") && !strings.HasSuffix(uri, ".txt") && !strings.HasSuffix(uri, ".gif") &&
		!strings.HasSuffix(uri, ".jpeg") && !strings.HasSuffix(uri, ".jpg") && !strings.HasSuffix(uri, ".css") {
		// 400 Bad Request
		fmt.Printf("[%d] Invalid uri format\n", requestId)
		conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n"))
		conn.Write([]byte("Content-Type: text/plain; charset=utf-8\r\n"))
		return
	}

	// check method
	if request.Method != "GET" && request.Method != "POST" {
		// 501 Not Implemented
		fmt.Printf("[%d] Invalid method: %s\n", requestId, request.Method)
		response.Headers["Content-Type"] = "text/plain"
		response.StatusCode = "501"
		write(conn, response)
		return
	}
	fmt.Printf("[%d] method ok.\n", requestId)

	// post, get dispatch.
	if request.Method == "POST" {
		handlePostMethod(request, response, conn, requestId)
	} else if request.Method == "GET" {
		handleGetMethod(conn, uri, response, requestId)
	}

	// json. json.NewDecoder(request.Body).Decode()
	// maybe for advanced part as a proxy. http.NewRequestWithContext(context.Background(), "GET", "http://"+conn.LocalAddr().String(), nil)

	return
}

func handleGetMethod(conn net.Conn, uri string, response *Response, requestId int) {
	fmt.Printf("[%d] into get method.\n", requestId)
	// find file
	fileData, err := os.ReadFile(uri[1:])
	if err != nil {
		// 404 not found
		fmt.Printf("[%d] Error reading file: %s", requestId, err)
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
	setContentType(response, ext)

	write(conn, response)
}
func handlePostMethod(request *http.Request, response *Response, conn net.Conn, requestId int) {
	fmt.Printf("[%d] into post method.\n", requestId)
	err := request.ParseMultipartForm(32 << 20)
	if err != nil {
		fmt.Printf("[%d] Error parsing multipart form: %s", requestId, err)
		response.StatusCode = "500"
		write(conn, response)
		return
	}
	fileMap := request.MultipartForm.File
	for k, v := range fileMap {
		fmt.Printf("filemap, k: %s, v_filename: %s\n", k, v[0].Filename)
		multipartFile := v[0]
		filename := multipartFile.Filename
		sourceFD, err := multipartFile.Open()
		if err != nil {
			fmt.Printf("[%d] Error opening source file: %s", requestId, err)
			response.StatusCode = "500"
			write(conn, response)
			return
		}
		defer sourceFD.Close()

		targetFD, err := os.Create("files/" + filename)
		if err != nil {
			fmt.Printf("[%d] Error creating target file: %s", requestId, err)
			response.StatusCode = "500"
			write(conn, response)
			return
		}
		defer targetFD.Close()
		_, err = io.Copy(targetFD, sourceFD)
		if err != nil {
			fmt.Printf("[%d] Error copying file: %s", requestId, err)
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
