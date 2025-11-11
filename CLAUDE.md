# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This is a **TDA596 Distributed Systems** course repository containing multiple lab assignments implementing distributed systems concepts in Go. The repo contains work from a 3-person team with different branches per member.

## Branch Structure

- `main` - Base branch with lab READMEs and specifications
- `arv-lab1` - Arv's working branch (beginner level)
- `wills-lab1` - Will's implementation
- `xin_dev` - Xin's implementation

When working on Arv's branch, remember this is learning-focused code. Explain concepts thoroughly and build incrementally with testing at each step.

## Lab 1: HTTP Server & Proxy (Current Focus)

Located in `Lab1_HTTP-Server/`. See `DEVELOPMENT_LOG.md` for detailed development strategy and progress tracking.

### Current Progress (arv-lab1 branch)

**Completed (Session 1 - 2025-11-11):**
- ✅ TCP socket listener with infinite loop for multiple clients
- ✅ Manual HTTP request parsing using `strings.Split()` and `strings.Fields()`
- ✅ Extract method and path from HTTP request line
- ✅ File serving from `server/files/` directory using `os.ReadFile()`
- ✅ 404 Not Found error handling
- ✅ Support for .html, .txt, and .jpg files

**Completed (Session 2 - 2025-11-11):**
- ✅ Content-Type headers based on file extension
- ✅ File extension validation (.html, .txt, .gif, .jpeg, .jpg, .css only)
- ✅ HTTP method validation (GET/POST only, 501 for others)
- ✅ 400 Bad Request for invalid extensions
- ✅ **POST implementation for file uploads - COMPLETE**
  - HTTP body parsing with `strings.Index()`
  - File writing to disk with `os.WriteFile()`
  - Uploaded files accessible via GET
  - Full validation and error handling

**Completed (Session 3 - 2025-11-11):**
- ✅ **Command-line port argument - COMPLETE**
  - Port parsing from `os.Args[1]`
  - Validation with `strconv.Atoi()`
  - Usage message when port missing
  - Error message for invalid port
  - Graceful exit with `os.Exit(1)`
  - All tests passed (no port, invalid port, port 8080, port 3000)

**Current Implementation:**
- Using manual parsing (will refactor to `http.ReadRequest()` later for learning)
- 215 lines in `main.go` with 2 helper functions
- POST and GET fully functional and tested
- All HTTP status codes implemented (200, 400, 404, 501)
- Dynamic port from command-line with validation

**Next Steps:**
1. Concurrency control (max 10 goroutines) - **CRITICAL REQUIREMENT** (2-3h)
2. Refactor to functions
3. Refactor manual parsing to `http.ReadRequest()`
4. Implement proxy (8-10h, worth 7 points)

### Critical Requirements

**MUST handle concurrency:**
- Max 10 concurrent connections using goroutines
- Each client request spawns a new goroutine
- Implement semaphore pattern or channel-based limiting

**Allowed packages:**
- `net` for TCP networking (MUST use `net.Listen()`)
- `net/http` for **parsing only** (use `http.ReadRequest()`)
- Standard library: `bufio`, `os`, `io`, `path/filepath`

**FORBIDDEN:**
- `http.ListenAndServe()`, `http.Listen()`, `http.Serve()` - these trivialize the assignment
- Web frameworks like Gin, Martini, Echo
- Any proxy methods from `http` package

### Implementation Approach

Build in layers, testing after each:

1. **TCP socket** - Listen on command-line port, accept connections
2. **HTTP parsing** - Read and parse HTTP/1.1 requests
3. **GET implementation** - Serve files from `server/files/`
4. **Validation** - Check extensions (.html, .txt, .gif, .jpeg, .jpg, .css), return proper status codes
5. **POST implementation** - Accept file uploads, make them accessible via GET
6. **Concurrency control** - Enforce 10 connection limit
7. **Proxy** - Forward GET requests between client and server

### Status Codes Required

- `200 OK` - Successful request
- `400 Bad Request` - Invalid extension or malformed request
- `404 Not Found` - File doesn't exist
- `501 Not Implemented` - Unsupported HTTP method

### Content-Type Mapping

```
.html → text/html
.txt  → text/plain
.gif  → image/gif
.jpeg, .jpg → image/jpeg
.css  → text/css
```

## Building and Running

### Server
```bash
cd Lab1_HTTP-Server/server
go build -o http_server main.go
./http_server 8080
```

### Proxy
```bash
cd Lab1_HTTP-Server/proxy
go build -o proxy main.go
./proxy 9000
```

Binaries MUST be named `http_server` and `proxy` respectively. Port must be configurable via command-line argument.

## Testing Commands

```bash
# Basic GET
curl http://localhost:8080/test.html

# File not found
curl http://localhost:8080/nonexistent.html

# Invalid extension
curl http://localhost:8080/test.exe

# Unsupported method
curl -X DELETE http://localhost:8080/test.html

# POST file
curl -X POST http://localhost:8080/uploaded.txt --data "Hello World"

# Through proxy
curl -x localhost:9000 http://localhost:8080/test.html

# Connection test
telnet localhost 8080
```

## Development Philosophy for Arv's Branch

This student is learning distributed systems fundamentals. When assisting:

1. **Never write code automatically** - Guide with explanations, let them implement
2. **Test incrementally** - Build small pieces, verify each works before continuing
3. **Explain the "why"** - Don't just show syntax, explain networking concepts
4. **Update DEVELOPMENT_LOG.md** - Track progress, problems, and solutions
5. **Compare approaches** - Reference teammates' implementations for learning

The goal is understanding, not just working code.

## Other Labs (Future Work)

- **Lab 2**: MapReduce implementation (in `Lab2_MapReduce/`)
- **Lab 3**: Chord DHT protocol (in `Lab3_Chord/`)
- **Lab 4**: Build Raft consensus (optional, in `Lab4(Optional)_BuildRaft/`)

Lab 2 contains a complex Go codebase with Raft, KVRaft, and sharding implementations borrowed from MIT 6.5840.
