# Simple HTTP Server

This is a basic HTTP server written in Go, capable of serving static files and handling file uploads. It listens on a specified port and processes incoming HTTP GET and POST requests.

## Features

*   Serves static files (HTML, CSS, TXT, JPG, GIF) from the `files` directory.
*   Handles HTTP GET requests for files.
*   Handles HTTP POST requests for form data and multipart file uploads.
*   Saves uploaded files to the `files` directory.
*   Custom 404 Not Found page.
*   Error handling for bad requests (400), not found (404), and not implemented methods (501).
*   Concurrency control using a semaphore.

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

If you run http_server at root folder you maybe need to add a `/server/` into url. See [concurrency_test.sh](./server/concurrency_test.sh)
