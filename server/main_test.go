package main

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	go func() {
		start()
	}()

	m.Run()
}
func TestGET200(t *testing.T) {
	//go start()

	client := http.Client{}
	resp, err := client.Get("http://localhost:8082/files/tmp.txt")
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	assert.Equal(t, 200, resp.StatusCode)
}

// content-type: image/jpeg
func TestGET200_ContentType_JPEG(t *testing.T) {
	//go start()
	client := http.Client{}
	resp, err := client.Get("http://localhost:8082/files/a.jpg")
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "image/jpeg", resp.Header.Get("Content-Type"))
}

func TestGet400_BadRequest(t *testing.T) {
	//go start()

	client := http.Client{}
	resp, err := client.Get("http://localhost:8082/files/tmp.invalid")
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	assert.Equal(t, 400, resp.StatusCode)
}

func TestGET404_NotFound(t *testing.T) {
	//go start()

	client := http.Client{}
	resp, err := client.Get("http://localhost:8082/files/noexist.html")
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	assert.Equal(t, 404, resp.StatusCode)
}

func TestGet501_NotImplemented(t *testing.T) {
	//go start()
	client := http.Client{}
	resp, err := client.Head("http://localhost:8082/files/tmp.txt")
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	assert.Equal(t, 501, resp.StatusCode)
}

func TestPost200(t *testing.T) {
	//go start()
	// file that need to be uploaded
	file, err := os.Open("/Users/xinyi/Desktop/upload.html")
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fileWriter, err := writer.CreateFormFile("file", "upload.html")
	if err != nil {
		return
	}
	_, err = io.Copy(fileWriter, file)
	if err != nil {
		return
	}
	writer.Close()
	//io.Copy(ioutil.Discard, file)
	defer file.Close()

	req, err := http.NewRequest("POST", "http://localhost:8082/files/upload.html", body)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)

}

func TestPost500_NoBoundary(t *testing.T) {
	//go start()

	file, err := os.Open("/Users/xinyi/Desktop/upload.html")
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	defer file.Close()
	client := http.Client{}
	resp, err := client.Post("http://localhost:8082/files/n500.html", "multipart/form-data", file)
	if err != nil {
		fmt.Printf("%s\n", err)
		return

	}
	defer resp.Body.Close()
	assert.Equal(t, 500, resp.StatusCode)

}
