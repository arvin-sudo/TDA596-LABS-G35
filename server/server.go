package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	_ "path/filepath"
	"strconv"
	"strings"
)

func main() {
	// listen for incoming tcp connections
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic("Error listening: " + err.Error())
		return
	}

	fmt.Println("Listening on port 8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting:", err)
			continue
		}
		go handleConnection(conn)
	}

}

func handleConnection(conn net.Conn) {

	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Println("Connection close: " + err.Error())
			return
		}
	}(conn)
	fmt.Println("New connection from " + conn.RemoteAddr().String())
	fmt.Println("-------------------------")

	//
	reader := bufio.NewReader(conn)
	for {
		requestMessage, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Client closed connection: " + err.Error())
			return
		}
		// trim line spaces and \n chars
		requestMessage = strings.TrimSpace(requestMessage)

		fmt.Println("Request: " + requestMessage)

		if requestMessage == "" {
			continue
		}

		if checkRequestType(requestMessage) {

			if strings.HasPrefix(requestMessage, "GET") {
				// send file
				fileName := strings.TrimPrefix(requestMessage, "GET") // trim out fileName
				fileName = strings.TrimSpace(fileName)

				// check hasValidExtension

				if hasValidExtension(fileName) {
					sendFile(fileName, conn)
				} else {
					fmt.Println("Invalid file extension")
					_, err := conn.Write([]byte("ERR inValid file extension\n"))
					if err != nil {
						return
					}
				}

			} else if strings.HasPrefix(requestMessage, "POST") {
				// download file
				// POST filename.ext fileSize
				content := strings.TrimPrefix(requestMessage, "POST") // trim out fileName

				fields := strings.Fields(content)
				fileName := fields[0]
				fileSize, _ := strconv.Atoi(fields[1])

				if hasValidExtension(fileName) {
					downloadFile(fileName, conn, fileSize, reader)
				} else {
					fmt.Println("Invalid file extension.")
					_, err := conn.Write([]byte("ERR inValid file extension\n"))
					if err != nil {
						return
					}
				}

			} else {
				_, err := conn.Write([]byte("ERR Unknown command\n"))
				if err != nil {
					return
				}
			}
		} else {
			_, err := conn.Write([]byte("ERR (501) Method not Implemented\n"))
			if err != nil {
				return
			}
		}

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

// [GET, POST]
func checkRequestType(request string) bool {
	if strings.HasPrefix(request, "GET") ||
		strings.HasPrefix(request, "POST") {
		return true
	}
	return false
}

func sendFile(fileName string, connection net.Conn) {
	// fmt.Println(fileName)
	file, err := os.Open("files/" + fileName)

	// filename:  example.txt
	if err != nil {
		//fmt.Println("Error opening file: " + err.Error())
		connection.Write([]byte("ERR File not found\n"))
		panic("Error opening file: " + err.Error())
		return
	}

	fmt.Println(">>> filename:" + fileName)
	// close file
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Println("Error closing file: " + err.Error())
			return
		}
	}(file)

	// get file stat | size
	stat, _ := file.Stat()
	size := stat.Size()

	// write file size
	_, err = connection.Write([]byte(fmt.Sprintf("OK %d\n", size)))
	if err != nil {
		panic("Error writing to connection: " + err.Error())
		return
	}

	// copy file content to connection
	_, err = io.Copy(connection, file)
	if err != nil {
		panic("Error copying file: " + err.Error())
		return
	}
	// write confirmation (filename, size)
	fmt.Printf("Sent %s with %d bytes to client %s\n", fileName, size, connection.LocalAddr().String())
}

func downloadFile(fileName string, connection net.Conn, fileSize int, reader *bufio.Reader) {
	// create file
	// content, _ := reader.ReadString('\n')

	fmt.Println("fileName:" + fileName)
	fmt.Println(fileSize)
	// fmt.Println(content)

	// error here
	wrt, err := os.Create("files/downloaded_" + fileName)
	if err != nil {
		fmt.Println("Error creating file: " + err.Error())
		return
	}

	_, err = io.CopyN(wrt, reader, int64(fileSize))

	if err != nil {
		panic("Error writing file: " + err.Error())
		return
	}

	fmt.Printf("Downloaded %s with %d bytes from client %s\n",
		fileName, fileSize, connection.LocalAddr().String())
}
