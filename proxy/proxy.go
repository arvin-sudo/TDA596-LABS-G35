package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	PROXY_ADDR  = ":9000"
	SERVER_ADDR = "127.0.0.1:8080"
)

type ErrorResponse struct {
	Message string
	Code    int
	flag    bool
}

var (
	inValidFileExt   = ErrorResponse{"INVALID_FILE_EXTENSION", 400, false}
	methodNotAllowed = ErrorResponse{"METHOD_NOT_ALLOWED", 501, false}
	notError         = ErrorResponse{"NOT_ERROR", 0, true}
)

func main() {
	listener, err := net.Listen("tcp", PROXY_ADDR)
	if err != nil {
		log.Fatal(err)
	}

	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			log.Fatal(err)
			return
		}
	}(listener)

	log.Printf("Proxy listening on %s → forwarding to %s", PROXY_ADDR, SERVER_ADDR)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(clientConn net.Conn) {
	defer func(clientConn net.Conn) {
		err := clientConn.Close()
		if err != nil {
			log.Printf("Error closing client connection: %v", err)
			return
		}
	}(clientConn)
	clientReader := bufio.NewReader(clientConn)

	for {
		line, err := clientReader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("Client read error: %v", err)
			}
			return
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		log.Printf("[CLIENT COMMAND] %s", line)

		parts := strings.Fields(line)
		if len(parts) < 2 {
			clientConn.Write([]byte("ERR invalid command\n"))
			continue
		}

		method := strings.ToUpper(parts[0])
		fileName := parts[1]

		// REQUEST VALIDATION
		if !isValidMethod(method) {
			clientConn.Write([]byte(fmt.Sprintf("ERR %s %d\n", methodNotAllowed.Message, methodNotAllowed.Code)))
			continue
		}
		if !isValidExtension(fileName) {
			clientConn.Write([]byte(fmt.Sprintf("ERR %s %d\n", inValidFileExt.Message, inValidFileExt.Code)))
			continue
		}

		// CONNECT TO SERVER
		serverConn, err := net.Dial("tcp", SERVER_ADDR)
		if err != nil {
			clientConn.Write([]byte("ERR cannot connect to server\n"))
			continue
		}
		serverReader := bufio.NewReader(serverConn)

		// WRITE REQUEST TO SERVER + \n
		fmt.Fprintf(serverConn, "%s\n", line)

		// POST HANDLER (NOT NEEDED FOR PROXY SECTION)
		if method == "POST" {
			if len(parts) != 3 {
				clientConn.Write([]byte("ERR missing file size\n"))
				serverConn.Close()
				continue
			}
			fileSize, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				clientConn.Write([]byte("ERR invalid file size\n"))
				serverConn.Close()
				continue
			}
			log.Printf("Uploading file %s (%d bytes)", fileName, fileSize)
			_, err = io.CopyN(serverConn, clientReader, fileSize)
			if err != nil && !errors.Is(err, io.EOF) {
				log.Printf("Error forwarding POST file: %v", err)
				serverConn.Close()
				continue
			}
			log.Printf("POST file upload complete")
		}

		// FORWARD SERVER MESSAGES (e.g File Not Found err (only server can determine if a file exists)
		statusLine, err := serverReader.ReadString('\n')
		if len(statusLine) > 0 {
			_, _ = clientConn.Write([]byte(statusLine))
		}
		if err != nil && !errors.Is(err, io.EOF) {
			log.Printf("Server read error: %v", err)
			serverConn.Close()
			continue
		}

		// READING REQUEST FILE FROM SERVER
		if method == "GET" && strings.HasPrefix(statusLine, "OK") {
			parts := strings.Fields(statusLine)
			if len(parts) != 2 {
				log.Printf("Invalid server OK response")
				serverConn.Close()
				continue
			}
			size, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				log.Printf("Invalid file size from server")
				serverConn.Close()
				continue
			}
			_, err = io.CopyN(clientConn, serverReader, size)
			if err != nil && !errors.Is(err, io.EOF) {
				log.Printf("Error forwarding file content: %v", err)
			}
			log.Printf("Forwarded %d bytes of GET file data to client", size)
		}

		err = serverConn.Close()
		if err != nil {
			return
		}
		log.Printf("REQUEST COMPLETED: %s %s", method, fileName)
	}
}

// LAB only wants us to implement GET with proxy
func isValidMethod(method string) bool {
	return method == "GET" || method == "POST"
}

func isValidExtension(filename string) bool {
	validExtensions := []string{".html", ".js", ".css", ".txt", ".jpg", ".jpeg", ".gif"}
	ext := filepath.Ext(filename)
	if ext == "" {
		return false
	}
	for _, v := range validExtensions {
		if v == ext {
			return true
		}
	}
	return false
}
