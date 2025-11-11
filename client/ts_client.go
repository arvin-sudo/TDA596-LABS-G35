package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:9000")
	if err != nil {
		panic(err)
	}
	defer func(conn net.Conn) {
		err = conn.Close()
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	}(conn)

	reader := bufio.NewReader(os.Stdin)
	serverReader := bufio.NewReader(conn)

	for {
		fmt.Print("Enter request: ")
		request, _ := reader.ReadString('\n')
		request = strings.TrimSpace(request)

		if strings.HasPrefix(request, "POST") {
			sendFile(request, conn)
			return
		}

		filename := strings.TrimPrefix(request, "GET")

		if request == "" {
			continue
		}

		// Send GET request
		fmt.Fprintf(conn, request+"\n")

		// Read response header
		header, err := serverReader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading response:", err)
			return
		}
		header = strings.TrimSpace(header)

		if strings.HasPrefix(header, "ERR") {
			fmt.Println(header)
			continue
		}

		if strings.HasPrefix(header, "OK ") {
			sizeStr := strings.TrimPrefix(header, "OK ")
			size, _ := strconv.ParseInt(sizeStr, 10, 64)
			out, err := os.Create("files/downloaded_" + filename)
			if err != nil {
				fmt.Println("Error creating file:", err)
				return
			}

			// Copy exact number of bytes
			written, err := io.CopyN(out, serverReader, size)
			if err != nil {
				fmt.Println("Error receiving file:", err)
				out.Close()
				return
			}
			out.Close()

			fmt.Printf("Downloaded %s (%d bytes)\n", filename, written)
		}
	}
}

func download(request string, conn net.Conn) {

}

func sendFile(request string, conn net.Conn) {

	fileName := strings.TrimPrefix(request, "POST")
	fileName = strings.TrimSpace(fileName)
	file, err := os.Open("files/" + fileName)

	defer func(file *os.File, c net.Conn) {
		err := file.Close()
		if err != nil {
			fmt.Println("Error closing file:", err.Error())
			return
		}
		err = c.Close()
		if err != nil {
			fmt.Println("Error closing connection:", err.Error())
			return
		}

	}(file, conn)

	if err != nil {
		fmt.Println("Error opening file:", err)
	}

	stat, _ := file.Stat()
	size := stat.Size()

	_, err = conn.Write([]byte(fmt.Sprintf("POST %s %d\n", fileName, size)))
	if err != nil {
		fmt.Println("Error writing to connection: " + err.Error())
		return
	}

	// copy file content to connection
	_, err = io.Copy(conn, file)
	if err != nil {
		fmt.Println("Error writing file to server:", err)
		return
	}

	fmt.Printf("Sent %s with %d bytes to Server running at %s\n", fileName, size, conn.RemoteAddr())
}
