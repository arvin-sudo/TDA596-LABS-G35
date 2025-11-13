package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"testing"
)

func TestMain(m *testing.M) {
	go func() {
		start()
	}()

	m.Run()
}
func TestProxyGet200(t *testing.T) {
	//go start()

	proxyURL, _ := url.Parse("http://localhost:8081")
	proxy := http.ProxyURL(proxyURL)
	transport := &http.Transport{Proxy: proxy}
	client := &http.Client{Transport: transport}
	req, _ := http.NewRequest("GET", "http://google.com", nil)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}

	buffer := make([]byte, 1024)
	resp.Body.Read(buffer)
	fmt.Println(string(buffer))
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProxyPost501(t *testing.T) {
	//go start()

	proxyURL, _ := url.Parse("http://localhost:8081")
	proxy := http.ProxyURL(proxyURL)
	transport := &http.Transport{Proxy: proxy}
	client := &http.Client{Transport: transport}
	req, _ := http.NewRequest("POST", "http://google.com", nil)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	defer resp.Body.Close()
	assert.Equal(t, 501, resp.StatusCode)
}
