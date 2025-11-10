package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"path/filepath"
	"strings"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorBlue   = "\033[34m"
	colorYellow = "\033[33m"
)

// LogWriter wraps data direction with color and timestamps
type LogWriter struct {
	Prefix string
	Color  string
}

func (lw *LogWriter) Write(p []byte) (int, error) {
	ts := time.Now().Format("15:04:05.000")
	maxLen := 300
	if len(p) > maxLen {
		p = p[:maxLen]
	}
	log.Printf("%s%s [%s] %s%q%s\n",
		colorYellow, ts, lw.Prefix, lw.Color, p, colorReset)
	return len(p), nil
}

// INCOMING_CLIENT_ADDR
const listenAddr = "0.0.0.0:9000"

// SERVER_ADDR
const targetAddr = "127.0.0.1:8080"

func handleConnection(clientConn net.Conn) {
	defer func(clientConn net.Conn) {
		err := clientConn.Close()
		if err != nil {
			panic(err)
			return
		}
	}(clientConn)

	// Connect to the backend server
	serverConn, err := net.Dial("tcp", targetAddr)
	if err != nil {
		log.Printf("Failed to connect to backend: %v\n", err)
		return
	}
	defer func(serverConn net.Conn) {
		err := serverConn.Close()
		if err != nil {
			panic(err)
			return
		}
	}(serverConn)

	// handle request check [isValidRequest(hasValidRequestMethod, hasValidExtension)]

	// Pipe data both ways concurrently
	go func() {
		r := io.TeeReader(clientConn, &LogWriter{Prefix: "[CLIENT -> SERVER]", Color: colorYellow})
		_, _ = io.Copy(serverConn, r) // client → server
		serverConn.Close()
	}()

	r := io.TeeReader(serverConn, &LogWriter{Prefix: "[SERVER -> CLIENT]", Color: colorGreen})
	_, _ = io.Copy(clientConn, r) // server → client
}

func isValidRequest(requestString string) (string, bool) {
	// check request method
	if hasValidRequestType(requestString) {
		// fileName
		fileName := strings.TrimPrefix(requestString, "GET") // returns "fileName"
		if hasValidExtension(fileName) {
			return "VALID", true
		}
		return "ERR inValid Extension", false

	} else {
		return "ERR (501) Method not Implemented", false
	}
}

// html, txt, gif, jpeg, jpg, txt or css
func hasValidExtension(filename string) bool {
	fmt.Println("FileName in extension handler: ", filename)
	validExtension := [6]string{".html", ".js", ".css", ".js", ".css", ".txt"}
	ext := filepath.Ext(filename)
	fmt.Println(ext)
	if ext == "" {
		return false
	} else {
		for _, v := range validExtension {
			if v == ext {
				return true
			}
		}
	}
	return false
}

func hasValidRequestType(requestType string) bool {
	if strings.HasPrefix(requestType, "GET") ||
		strings.HasPrefix(requestType, "POST") {
		return true
	}
	return false
}

func main() {
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", listenAddr, err)
	}
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			panic(err)
			return
		}
	}(listener)

	log.Printf("Proxy listening on %s, forwarding to %s\n", listenAddr, targetAddr)

	for {
		conn, err := listener.Accept()
		// io.TeeReader(conn, &LogWriter{Prefix: "CONNECTION ACCEPTED", Color: colorGreen})
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go handleConnection(conn)
	}
}
