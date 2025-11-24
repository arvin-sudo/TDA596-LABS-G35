# Simple HTTP Server

This is a basic HTTP server written in Go, capable of serving static files and handling file uploads. It listens on a specified port and processes incoming HTTP GET and POST requests.

## Prerequisites
* Go 1.25 or higher
* Docker (optional)

## Features

*   Serves static files (HTML, CSS, TXT, JPG, GIF) from the `files` directory.
*   Handles HTTP GET requests for files.
*   Handles HTTP POST requests for form data and multipart file uploads.
*   Saves uploaded files to the `files` directory.
*   Custom 404 Not Found page.
*   Error handling for bad requests (400), not found (404), and not implemented methods (501).
*   Concurrency control using a semaphore.

## API Endpoints
### GET /files/{filename}
Retreive static files from the 'files' directory.

Supported file extensions according to the specifications:
* `.html` - Content-Type: `text/html`
* `.txt` - Content-Type: `text/plain`
* `.gif` - Content-Type: `image/gif`
* `.jpeg`, `.jpg` - Content-Type: `image/jpeg`
* `.css` - Content-Type: `text/css`

Response codes:
* 200 OK - File retrieved successfully
* 400 Bad Request - Unsupported file extension
* 404 Not Found - File does not exist
* 500 Internal Server Error - Error reading file
* 501 Not Implemented - HTTP method not supported

### POST /files/{filename}
Handles file uploads using multipart form data. Saves uploaded files to the 'files' directory.

Response Codes:
* 200 OK - File uploaded successfully
* 400 Bad Request - Invalid request format
* 500 Internal Server Error - Error processing upload

## How to Run
### Running locally
```bash
go build -o server/http_server ./server
```
```bash
./server/http_server 8080
```

### Running by docker container
#### Building and pushing (linux)
```bash
docker build -f server/Dockerfile --tag docker-server:1.1 .
docker tag docker-server:1.1 daryl1104/docker-server:1.1
docker push daryl1104/docker-server:1.1
```
#### Building and pushing (mac)
We need to push not only arm64, but also amd64 to Docker hub, because AWS EC2 is linux-based system in our case.
```bash
docker buildx build -f server/Dockerfile --platform linux/amd64,linux/arm64 -t daryl1104/docker-serveer:1.1 --push .
```

#### Running
```bash
docker run -d --workdir=/app/server -p 8080:8080 daryl1104/docker-server:1.1
```


## Design and Implementation
### Data structures
* `Response`: store http status code, body content as byte array, http headers.
* `httpCodeMap`: a simple map to get status text from code.

### Flow
1. Setting a tcp listener with a specific port(default 8080, but can change as argument by running binary file.)
2. Continuously accepting connection from clients and using goroutine to asynchronous process requests. Using channel to simulate a semaphore to control rate limit or request capability. Generating a randomly request id to make it easier to track in log.
3. Processing in goroutine:
    1. Reading raw data from connection. Reading data and storing into a buffer.
    2. Parsing to standard http request. Using `http.ReadRequest` from `net/http` to parsing as http request.
    3. Validating http method and file extension. Using path to compare extension.
    4. Dispatching to `handlePostMethod` or `handleGetMethod` accoding to http method.
       * GET: 
         1. Reading file content and construct http response. Using `os.ReadFile` to read content and setting to response body.
       * POST:
         1. Formatting normal form request or binary file form request according to `content-type` ("application/x-www-form-urlencoded" means normal form, "multipart/form-data" means posting file).
         2. Reading file content and storing in `files` folder with the same name of this file.
    5. Writing in http standard message format.

## Implementation Challenges

### Connection Reading Deadlock
Initial implementation used for loop reading data from TCP Connection. Client applications like Postman do not close connections immediately after sending request data, causing `conn.Read()` to block waiting for FIN packet. This resulted in deadlock where goroutine waits indefinitely. Solution changed to single `conn.Read()` call with large buffer (20MB) to read entire request in one operation. Buffer explicitly set to `nil` after parsing to allow garbage collection and prevent memory exhaustion under high load.

### Concurrect Connection Management
Lab requirement specifies maximum 10 concurrent connections. Implemented using semaphore pattern with buffered channel of size 10. Each accepted connection sends value to channel before spawning goroutine, blocking if channel full. Connection completion receives from channel, freeing slot for next request.

### HTTP Protocol Parsing
Initial approach manually parsed HTTP headers using string splitting. Refactored to use `http.ReadRequest()` from standard library for robust parsing. Request data buffered using `bufio.NewReader(bytes.NewReader(buffer))` to provide Reader interface. It eliminated custom parsing logic and handles edge cases in HTTP protocol specification automatically.

### File Upload Buffer Sizing
GET requests for small files work with default buffer sizes. POST requests with multipart form data for image uploads require larger buffers to accommodate file content. Buffer size increased from 4KB to 20MB. Memory released explicitly after request parsing to prevent heap space exhaustion with many concurrent uploads.


## Testing
### Functionality:
Several options:
1. The `main_test.go` file contains unit tests, running by ```go test ./server```
2. Running ```curl```
   1. GET: ```curl -i http://localhost:8080/files/test.txt```
   2. POST: ```curl -i -F "vv=@/Users/xinyi/Desktop/tmp.txt" http://localhost:8080/files/tmp.txt```
3. Mannually sending requests from Postman or Chrome.

### Concurrency:
Uncomment line 70 ```time.Sleep(10 * time.Second)``` in `main.go`, recompile and rerun. Running ```./server/concurrency_test.sh``` to test.

If you run http_server at root folder you maybe need to add a `/server/` into url. See [concurrency_test.sh](serveroncurrency_test.sh)
