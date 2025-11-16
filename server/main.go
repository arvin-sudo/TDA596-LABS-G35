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
	BBody      []byte
}

func main() {
	start()
}

func start() {
	rawPort := "8080"
	if len(os.Args) > 1 {
		rawPort = os.Args[1]
	}
	if _, err := strconv.Atoi(rawPort); err != nil {
		rawPort = "8080"
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
		go handleTCPConnection(conn, sem, rand.IntN(100))
	}
}

func handleTCPConnection(conn net.Conn, sem chan int, requestId int) {
	// slow down process to test concurrency capability.
	// time.Sleep(10 * time.Second)

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
	fmt.Printf("[%d] Accepted connection.\n", requestId)

	response := &Response{}
	response.Headers = make(map[string]string)
	response.StatusCode = "200"

	/**
	1. read data from tcp connection
	*/
	// 20M. make bigger for uploading files such as image.
	buffer := make([]byte, 20*1024*1024)
	// trap: 1. postman reused tcp connection, so one clicked will trigger into two go routines.
	// 2. don't use for loop to receive data, the client will not close connection, conn.Read() will get block waiting a close request(FIN). then deadlock.
	n, err := conn.Read(buffer)
	fmt.Printf("[%d] n: %d err: %v\n", requestId, n, err)
	if err != nil {
		fmt.Printf("[%d] Error reading from connection: %s", requestId, err)
		response.StatusCode = "500"
		write(conn, response)
		return
	}

	fmt.Printf("[%d] buffer is readed.\n", requestId)

	/**
	2. parse data into http protocol message
	*/
	request, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(buffer)))
	// in order to release heap space even though it rely on GC, but it's at most what we could do.
	// in case many requests exhausted heap space crash down service.
	buffer = nil
	if err != nil {
		// if parsing error, then send back, 400 Bad Request
		fmt.Printf("[%d] Error parsing request: %s", requestId, err)
		response.Headers["Content-Type"] = "text/html"
		response.StatusCode = "400"
		response.BBody, _ = file2Bytes(requestId, "files/404.html")
		write(conn, response)
		return
	}

	fmt.Printf("[%d] construct request ok.\n", requestId)

	/**
	3. validate this request if formatted correctly?
	*/
	allowed := validate(requestId, request, conn, response)
	if !allowed {
		return
	}

	/**
	4. dispatch to different processing method.
	*/
	if request.Method == "POST" {
		handlePostMethod(request, response, conn, requestId)
	} else if request.Method == "GET" {
		path := request.URL.Path
		handleGetMethod(conn, path, response, requestId)
	}

	return
}

func validate(requestId int, request *http.Request, conn net.Conn, response *Response) bool {
	// check extension
	path := request.URL.Path
	fmt.Printf("[%d] path: %s\n", requestId, path)
	if !strings.HasSuffix(path, ".html") && !strings.HasSuffix(path, ".txt") && !strings.HasSuffix(path, ".gif") &&
		!strings.HasSuffix(path, ".jpeg") && !strings.HasSuffix(path, ".jpg") && !strings.HasSuffix(path, ".css") {
		// 400 Bad Request
		fmt.Printf("[%d] Invalid uri format\n", requestId)
		response.Headers["Content-Type"] = "text/html"
		response.StatusCode = "400"
		write(conn, response)
		return false
	}

	// check method
	if request.Method != "GET" && request.Method != "POST" {
		// 501 Not Implemented
		fmt.Printf("[%d] Invalid method: %s\n", requestId, request.Method)
		response.Headers["Content-Type"] = "text/html"
		response.StatusCode = "501"
		write(conn, response)
		return false
	}
	fmt.Printf("[%d] Pass validation.\n", requestId)
	return true
}

func handleGetMethod(conn net.Conn, uri string, response *Response, requestId int) {
	fmt.Printf("[%d] Get into get method.\n", requestId)
	fileData, err := file2Bytes(requestId, uri[1:])
	if err != nil {
		// 404 not found
		fmt.Printf("[%d] Error reading file: %s", requestId, err)
		response.Headers["Content-Type"] = "text/html"
		response.StatusCode = "404"
		response.BBody, _ = file2Bytes(requestId, "files/404.html")
		write(conn, response)
		return
	}

	// set body
	response.BBody = fileData

	// set headers. (only for content-type right now)
	ext := filepath.Ext(uri)
	setContentType(response, ext)

	write(conn, response)
}
func handlePostMethod(request *http.Request, response *Response, conn net.Conn, requestId int) {
	fmt.Printf("[%d] Get into post method.\n", requestId)
	contentType := request.Header.Get("Content-Type")

	if contentType == "" || strings.Contains(contentType, "application/x-www-form-urlencoded") {
		// normal form
		err := request.ParseForm()
		if err != nil {
			response.StatusCode = "500"
			write(conn, response)
			return
		}
		// no logic, just return 200 OK
		write(conn, response)
	} else if strings.Contains(contentType, "multipart/form-data") {
		// parse request body as multipart/form-data
		err := request.ParseMultipartForm(32 << 20)
		if err != nil {
			fmt.Printf("[%d] Error parsing multipart form: %s", requestId, err)
			response.StatusCode = "500"
			write(conn, response)
			return
		}
		fileMap := request.MultipartForm.File
		// save files.
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

func file2Bytes(requestId int, filename string) ([]byte, error) {
	fileData, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("[%d] Error reading file: [%s], err: %s\n", requestId, filename, err)
		return []byte("<h1>ERROR</h1>"), err
	}
	return fileData, nil
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
