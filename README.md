# TDA596 Lab 1 - HTTP Server, Proxy and Cloud - Group 35
This repository contains implementations of an HTTP server and HTTP proxy written in Go. The server handles GET and POST requests for file serving and uploads. The proxy forwards GET requests to remote server.

## Prerequisites
* Go 1.25 or higher
* Docker (optional)
* 'curl' for testing

## This includes two services: 
* [http proxy](./README_proxy.md)
  * A simple http proxy.
    * only support GET method.
* [http server](./README_server.md)
  * A simple http server.
    * support GET,POST method.
    * support .html, .txt, .gif, .jpeg, .jpg, .css