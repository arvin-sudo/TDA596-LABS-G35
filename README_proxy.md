# HTTP Proxy Server

This is a simple HTTP proxy server written in Go. It listens on a specified port, accepts incoming HTTP GET requests, and forwards them to the target server. It then relays the response back to the client.

## Prerequisites
* Go 1.25 or higher
* Docker (optional)

## Features

*   Listens for incoming TCP connections.
*   Parses HTTP requests.
*   Forwards GET requests to the target server.
*   Relays server responses back to the client.
*   Handles 400 Bad Request for malformed requests.
*   Handles 501 Not Implemented for non-GET methods.
*   Concurrency control using a semaphore to limit active connections.

## Proxy Mechanism
Acting as intermediary between client and server. Client sends request to proxy with target URL. Proxy parses request and extracts target server address. Proxy creates new request to target server. Proxy forwards request and receives response. Proxy relays response back to client.

## How to Run
### Running locally
```bash
go build -o proxy/proxy ./proxy
```
```bash
./proxy/proxy 8081
```

### Running by docker container
#### Building and pushing (linux)
```bash
docker build -f proxy/Dockerfile --tag docker-proxy:1.1 .
docker tag docker-proxy:1.1 daryl1104/docker-proxy:1.1
docker push daryl1104/docker-proxy:1.1
```
#### Building and pushing (mac)
We need to push not only arm64, but also amd64 to Docker hub, because AWS EC2 is linux-based system in our case.
```bash
docker buildx build -f proxy/Dockerfile --platform linux/amd64,linux/arm64 -t daryl1104/docker-proxy:1.1 --push .
```

#### Running
```bash
docker run -d --workdir=/app/proxy -p 8081:8081 daryl1104/docker-proxy:1.1
```

## Design and Implementation
### Data Structures
* `Response`: store http status code, body content as byte array, http headers.
* `httpCodeMap`: a simple map to get status text from code. 

### Flow
1. Setting a tcp listener with a specific port(default 8081, but can change as argument by running binary file.)
2. Continuously accepting connection from clients and using goroutine to asynchronous process requests. Using channel to simulate a semaphore to control rate limit or request capability. Generating a randomly request id to make it easier to track in log.
3. Processing in goroutine:
   1. Reading raw data from connection. Reading data and storing into a buffer.
   2. Parsing to standard http request. Using `http.ReadRequest` from `net/http` to parsing as http request.
   3. Validating http method.
   4. Initializing a new http client and a new http request, copying headers and body content to it.
   5. Invoking this http request and waiting for the response.
   6. Copying http status code, body to our own response.
   7. Writing response in http standard message format.
      
## Implementation Challenges
### Request Forwarding Architecture
Proxy acts as both HTTP client and HTTP server simultaneously. Incoming client connection handled as server using `http.ReadRequest()`. Outgoing connection to target server uses `http.DefaultClient.Do()` with new request constructed from parsed client request. Response from target server copied to original client connection.

### Header Management
Direct forwarding of all headers caused issues with proxy-specific headers. Header `Proxy-Connection` removed before forwarding to target server as it is proxy-control header not understood by origin servers. All other headers copied from client request to target request. Response headers copied back to client without modification.

### Concurrency Control
Similar to server implementation, proxy limits concurrent connections using semaphore pattern. Buffered channel of size 10 controls number of simultaneous proxy operations. Each client connection acquires semaphore slot before processing, releases on completion.

### Method Restriction
Proxy implements only GET method according to lab advanced part specification. POST and PUT methods require forwarding request body from client to target server, adding complexity of streaming potentially large data. Current implementation returns `501 Not Implemented` for non-GET methods. GET method sufficient to demonstrate proxy forwarding mechanism without body handling complexity.




## Testing
Several options:
1. The `main_test.go` file contains unit tests, running by ```go test ./proxy```
2. Running ```curl -X GET http://httpbin.org/get -x localhost:8081``` setting proxy to test.
3. Using browser and proxy plugin to test.

## Limitations
* Supports GET requests only. Other HTTP methods return `501 Not Implemented`.
* No HTTPS support.
* No response caching.
* Request buffer limited to 4KB.
* Header `Proxy-Connection` removed before forwarding to target server.