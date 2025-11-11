# Lab 1 - HTTP Server & Proxy Development Log

**Kurs:** TDA596 - Distributed Systems
**Labmedlemmar:** Arv (beginner), Will, Xin
**Startdatum:** 2025-11-11

---

## Projektöversikt

### Vad bygger vi?
Lab 1 handlar om att bygga **grundläggande nätverkskommunikation** från scratch i Go. Vi lär oss hur webben fungerar "under huven" - hur servrar och klienter kommunicerar.

### Två huvuddelar:

#### Del 1: HTTP Server (10 poäng)
En webserver som kan:
- ✅ Ta emot HTTP-förfrågningar från klienter
- ✅ Skicka tillbaka filer (HTML, CSS, txt, bilder)
- ✅ Ta emot filer från klienter (POST/upload)
- ✅ Hantera flera klienter samtidigt (max 10 concurrent connections)
- ✅ Returnera korrekta HTTP-statuskoder (200, 400, 404, 501)

#### Del 2: HTTP Proxy (7 poäng)
En proxy som:
- ✅ Sitter mellan klient och server
- ✅ Vidarebefordrar GET-requests
- ✅ Returnerar svar från server till klient
- ✅ Möjliggör cachning, filtrering, anonymitet

---

## Hur fungerar HTTP?

### Request (från klient till server):
```
GET /test.html HTTP/1.1
Host: localhost:8080
```

**Betydelse:**
- `GET` = Jag vill ha en fil
- `/test.html` = Vilken fil
- `HTTP/1.1` = Protokollversion

### Response (från server till klient):
```
HTTP/1.1 200 OK
Content-Type: text/html
Content-Length: 123

<html><body>Hello!</body></html>
```

**Betydelse:**
- `200 OK` = Lyckades
- `Content-Type` = Typ av innehåll
- Efter blank rad kommer själva innehållet

---

## Vår Utvecklingsstrategi

Vi bygger **iterativt** (steg för steg) och **testar efter varje steg**.

### Lager 1: TCP-anslutning (Grunden)
**Vad:** Öppna en socket, lyssna på port
**Varför:** Grunden för all nätverkskommunikation
**Test:** `telnet localhost 8080`

### Lager 2: Läsa HTTP-requests
**Vad:** Läs och parsa inkommande HTTP-meddelanden
**Varför:** Vi måste förstå vad klienten vill
**Test:** Skicka request, logga vad vi tar emot

### Lager 3: Svara med filer (GET)
**Vad:** Läs fil från disk, skicka med HTTP-response
**Varför:** Huvudfunktionen för en webserver
**Test:** `curl http://localhost:8080/test.html`

### Lager 4: Validering & Felhantering
**Vad:** Kontrollera file extensions, hantera fel
**Varför:** Säkerhet och korrekt protokoll
**Test:** Testa ogiltiga requests

### Lager 5: Ta emot filer (POST)
**Vad:** Spara filer som klienten skickar
**Varför:** Tvåvägskommunikation
**Test:** Upload fil, sedan GET den

### Lager 6: Concurrency
**Vad:** Goroutines + semaphore (max 10)
**Varför:** Hantera många klienter samtidigt
**Test:** Många samtidiga anslutningar

### Lager 7: Proxy
**Vad:** Vidarebefordra requests mellan klient och server
**Varför:** Avancerad del av labben
**Test:** `curl -x proxy_ip:port server_ip:port/file`

---

## Jämförelse med labpartners

### Will's approach:
- ✅ Enklare custom protokoll (inte full HTTP)
- ✅ Format: `GET filename.txt` (inte `/filename.txt HTTP/1.1`)
- ✅ Lättare att förstå, men inte enligt spec

### Xin's approach:
- ✅ Riktig HTTP/1.1 implementation
- ✅ Använder `http.ReadRequest()` från Go
- ✅ Mer professionell, lite svårare

### Vår approach:
- ✅ Bygger som Xin (riktig HTTP)
- ✅ Men går igenom VARJE steg med förklaringar
- ✅ Jag skriver koden själv med guidning

---

## Utvecklingslogg

### 2025-11-11 - Session 1: TCP Server & Manual HTTP Parsing

#### Status: 🟢 Lager 1-3 Klara!

**Vad vi implementerat:**
- ✅ TCP socket listener på port 8080
- ✅ Accept multiple client connections (infinite loop)
- ✅ Read HTTP requests från klienter
- ✅ Manuell HTTP parsing med `strings.Split()` och `strings.Fields()`
- ✅ Extrahera method och path från request line
- ✅ Läsa filer från disk med `os.ReadFile()`
- ✅ Servera filinnehåll till klienter
- ✅ 404 Not Found för saknade filer
- ✅ Fungerande för .html, .txt, och .jpg filer

**Tekniska koncept som lärts:**
1. **Parsing** - Att tolka och strukturera rå text till användbar data
2. **Variable scope** - Variabler måste deklareras utanför `{}` för att användas senare
3. **`:=` vs `=`** - `:=` skapar ny variabel, `=` sätter värde på existerande
4. **Binary vs text data** - Bilder kan serveras men inte visas i terminal
5. **HTTP response format** - Status line + headers + blank line + body

**Problem & lösningar:**
| Problem | Lösning |
|---------|---------|
| `path` undefined efter if-sats | Deklarera `var path string` före if-satsen |
| Server dog efter första klienten | Flytta `conn.Close()` till slutet av loopen |
| Bilden laddas ner istället för visas | Content-Type är fel (fixas nästa steg) |
| `files/` folder placement confusion | Flyttade till `server/files/` för korrekt working directory |

**Testresultat:**
```bash
✅ curl localhost:8080/test.html    → HTML visas korrekt
✅ curl localhost:8080/test.txt     → Text visas korrekt
✅ curl localhost:8080/image.jpg    → Binär data skickas (fungerar i webbläsare)
✅ curl localhost:8080/notfound     → 404 Not Found
```

**Kod-struktur:**
- 115 rader i `main.go`
- Allt i `main()` funktion (refaktoreras senare)
- Manuell parsing med standard library (`strings` package)

**Nästa steg:**
1. Implementera Content-Type headers baserat på file extension
2. Validera file extensions (.html, .txt, .gif, .jpeg, .jpg, .css only)
3. Validera HTTP methods (endast GET och POST)
4. Returnera 400 Bad Request för ogiltiga extensions
5. Returnera 501 Not Implemented för andra metoder

**Anteckningar:**
- Valde manuell parsing först för att förstå HTTP-protokollet
- Kommer refaktorera till `http.ReadRequest()` senare
- Incremental testing fungerar utmärkt - varje litet steg verifieras
- Go's error handling med `if err != nil` är tydlig och konsekvent

---

### 2025-11-11 - Session 2: POST Implementation & Full HTTP Validation

#### Status: 🟢 Lager 4-5 Klara! Server funktionellt komplett för GET/POST!

**Vad vi implementerat:**
- ✅ Content-Type headers baserat på file extension (getContentType funktion)
- ✅ File extension validation (isValidExtension funktion med whitelist)
- ✅ HTTP method validation (endast GET och POST tillåtna)
- ✅ 400 Bad Request för ogiltiga file extensions
- ✅ 501 Not Implemented för andra HTTP methods (DELETE, PUT, etc)
- ✅ **POST implementation - KOMPLETT!**
  - Body parsing med `strings.Index()` för att hitta `\r\n\r\n`
  - Extrahera body content efter headers
  - Validera file extension före sparning
  - Spara filer till disk med `os.WriteFile()`
  - Returnera 200 OK eller lämpliga felkoder
  - Uppladdade filer blir automatiskt tillgängliga via GET

**Tekniska koncept som lärts:**
1. **HTTP body parsing** - Hitta var headers slutar och body börjar (`\r\n\r\n`)
2. **String slicing** - `requestStr[bodyStartIndex:]` för att extrahera substring
3. **File I/O** - `os.WriteFile()` med permissions (0644)
4. **Request/Response cycle** - POST sparar fil, GET läser samma fil
5. **Whitelisting** - Säkerhetsansats att bara tillåta kända säkra extensions
6. **Error handling** - 400, 404, 500, 501 statuskoder för olika fel
7. **Control flow** - `if/else if` för att separera POST och GET handling
8. **Connection management** - `conn.Close()` i rätt fall, `continue` för att återgå till loop

**Problem & lösningar:**
| Problem | Lösning |
|---------|---------|
| GET-block saknade avslutande `}` | Lade till korrekt `} else if method == "GET" { ... }` struktur |
| Onåbar kod efter GET/POST | Tog bort duplicate prints som aldrig kördes (rad 215-222) |
| Body parsing förståelse | Förklarade HTTP struktur: headers → blank line → body |
| `filepath` variable shadowing | Använder samma variabel-namn i både POST och GET (OK i olika scopes) |

**Testresultat - ALLA TESTER KLARADE:**
```bash
✅ POST /uploaded.txt        → "File uploaded successfully"
✅ GET /uploaded.txt         → "Hello from POST test!" (filen vi laddade upp)
✅ GET /test.html            → HTML-innehåll returneras korrekt
✅ POST /malware.exe         → "400 Bad Request: Invalid file extension"
✅ GET /missing.html         → "File Not Found" (404)
```

**Kod-struktur:**
- 215 rader i `main.go` (ner från 224 efter cleanup)
- 2 helper functions: `getContentType()` och `isValidExtension()`
- POST block: rad 127-179
- GET block: rad 180-213
- Strukturerad if/else for clean separation of concerns

**Jämförelse med xin_dev (labpartner):**
- ✅ POST implementation - VI har det nu, xin_dev har INTE
- ✅ GET implementation - Båda lika
- ✅ Validation (extensions, methods) - Båda lika
- ❌ Concurrency control - xin_dev HAR (semaphore pattern), vi saknar (NEXT STEP)
- ❌ Proxy implementation - xin_dev HAR (141 lines), vi saknar
- ⚠️ Parsing approach - Vi använder manual (`strings.Split`), xin_dev använder `http.ReadRequest()`
- ⚠️ Code organization - xin_dev har Response struct, vi har inline responses

**Tidsestimation till feature parity med xin_dev:**
- Command-line port argument: 30 min (lätt fix)
- Concurrency control: 3h (kritiskt krav, semaphore pattern)
- Refactor till http.ReadRequest(): 1.5h (lärande)
- Response struct pattern: 1h (kod-kvalitet)
- Proxy implementation: 8-10h (största featuren)
- **Total:** ~15-18h till full paritet

**Nästa steg (prioriterat):**
1. ⏭️ Command-line port argument (snabb win, 30 min)
2. ⏭️ Concurrency control (OBLIGATORISKT krav, 3h)
3. ⏭️ Refactor till `http.ReadRequest()` (lärande + best practice)
4. ⏭️ Proxy implementation (8-10h, värd 7 poäng)

**Anteckningar:**
- Manual HTTP parsing gav djup förståelse av protokollet
- POST var enklare än förväntat tack vare tydlig struktur
- Testdriven utveckling fungerar perfekt - små steg, testa ofta
- `go run main.go` för snabb utveckling, `go build` för inlämning
- Alla core features för en basic HTTP server är nu klara!
- Nästa stora utmaning: Concurrency (goroutines + semaphore)

---

### 2025-11-11 - Session 3: Command-Line Port Argument

#### Status: 🟢 Steg 1 Klart! Dynamisk port från command-line!

**Vad vi implementerat:**
- ✅ Command-line argument parsing med `os.Args[1]`
- ✅ Port validation med `strconv.Atoi()`
- ✅ Usage message när port saknas
- ✅ Error message för ogiltig port
- ✅ Graceful exit med `os.Exit(1)` på fel

**Tekniska koncept som lärts:**
1. **Command-line arguments** - `os.Args` är en slice med alla argument
   - `os.Args[0]` = programnamn (`./http_server`)
   - `os.Args[1]` = första argumentet (vår port)
2. **Array length checking** - `len(os.Args) < 2` kollar om användaren gav argument
3. **String to integer conversion** - `strconv.Atoi()` returnerar `(int, error)`
4. **Blank identifier** - `_` används för att ignorera returvärden vi inte behöver
5. **Error checking pattern** - `if err != nil` för att kolla om något gick fel
6. **Exit codes** - `os.Exit(1)` för fel, `os.Exit(0)` för success
7. **`os.Exit()` vs `panic()`** - `os.Exit()` för user errors (renare), `panic()` för bugs

**Problem & lösningar:**
| Problem | Lösning |
|---------|---------|
| Kompileringsfel: `port` undefined på rad 59 | Lade till `var port string` declaration före if-sats, ELLER tog bort redundant `port := "8080"` |
| Förstå `_, err :=` syntax | Förklaring: `_` ignorerar numret från `Atoi()`, vi vill bara kolla error |
| Val mellan strict vs fallback approach | Valde strict (kräver port) för tydlighet och learning |

**Testresultat - ALLA TESTER KLARADE:**
```bash
✅ ./http_server              → "Usage: ./http-server <port>" (exit 1)
✅ ./http_server abc          → "Invalid port number: abc" (exit 1)
✅ ./http_server 8080         → "Server is listening on port 8080"
   curl localhost:8080/test.html → HTML returneras korrekt
✅ ./http_server 3000         → "Server is listening on port 3000"
   curl localhost:3000/test.txt  → Text returneras korrekt
```

**Kod-struktur:**
- 215 rader i `main.go` (ingen ändring i total radantal)
- Lade till `strconv` import
- Port validation block: rad 52-65
- Tog bort redundant `port := "8080"` declaration
- Lade till kommentarer för PORT SETUP AND VALIDATION sektion

**Jämförelse med labpartners:**
| Feature | Will | Xin | Arv (vi) |
|---------|------|-----|----------|
| Port argument | ❌ Hårdkodad `:8080` | ✅ Med default fallback | ✅ Strict (måste ange) |
| Validation | ❌ Ingen | ✅ `strconv.Atoi()` | ✅ `strconv.Atoi()` |
| Error handling | N/A | `panic()` | `os.Exit(1)` |
| HTTP protocol | ❌ Custom | ✅ HTTP/1.1 | ✅ HTTP/1.1 |
| POST support | ✅ Ja | ❌ Nej | ✅ Ja |
| Concurrency | ✅ Goroutines | ✅ Goroutines + semaphore | ❌ Saknas (NEXT) |

**Slutsats:** Vi har nu bättre port handling än Will, och lika bra som Xin (men mer strict). Will's implementation kommer ha problem vid demo eftersom den inte följer HTTP-standarden!

**Nästa steg (prioriterat):**
1. ⏭️ **Concurrency control** (KRITISKT KRAV) - 2-3h
   - Refaktorera till `handleClient()` funktion
   - Lägg till goroutines för varje connection
   - Implementera semaphore pattern (buffered channel med cap 10)
2. ⏭️ Refactor till `http.ReadRequest()` (1.5h)
3. ⏭️ Proxy implementation (8-10h, värd 7 poäng)

**Anteckningar:**
- Command-line parsing var enklare än förväntat
- `strconv.Atoi()` är standard-metoden för att validera numeriska strings
- Strict approach (ingen default) tvingar explicit port - bra för learning
- Test-driven development fortsätter att fungera perfekt
- Redo för nästa stora steg: Concurrency!

---

## Testplan

### Manuella tester vi kommer köra:

#### Steg 1: TCP connection
```bash
telnet localhost 8080
```
**Förväntat:** Connection accepted

#### Steg 2: HTTP GET
```bash
curl http://localhost:8080/test.html
```
**Förväntat:** HTML-innehåll returneras

#### Steg 3: File not found
```bash
curl http://localhost:8080/nonexistent.html
```
**Förväntat:** 404 Not Found

#### Steg 4: Invalid extension
```bash
curl http://localhost:8080/test.exe
```
**Förväntat:** 400 Bad Request

#### Steg 5: Unsupported method
```bash
curl -X DELETE http://localhost:8080/test.html
```
**Förväntat:** 501 Not Implemented

#### Steg 6: POST file
```bash
curl -X POST http://localhost:8080/uploaded.txt --data "Hello World"
```
**Förväntat:** File saved, 200 OK

#### Steg 7: Through proxy
```bash
curl -x localhost:9000 http://localhost:8080/test.html
```
**Förväntat:** Content via proxy

---

## Tekniska koncept att förstå

### 1. TCP/IP
- Transport layer protocol
- Reliable, connection-oriented
- Port + IP = unique endpoint

### 2. Goroutines
- Lightweight threads in Go
- `go functionName()` starts concurrent execution
- Need synchronization for shared resources

### 3. HTTP Status Codes
- `200 OK` - Success
- `400 Bad Request` - Client error (malformed request)
- `404 Not Found` - Resource doesn't exist
- `501 Not Implemented` - Server doesn't support method

### 4. Concurrency Control
- Semaphore pattern: limit concurrent goroutines
- Use channels or sync primitives
- Max 10 concurrent connections requirement

---

## Viktiga Go packages

```go
import (
    "net"           // TCP networking
    "net/http"      // HTTP parsing (allowed for parsing only)
    "bufio"         // Buffered I/O
    "os"            // File operations
    "io"            // I/O utilities
)
```

**Viktigt:** Vi får INTE använda `http.ListenAndServe()` - det trivialiserar uppgiften!

---

## Problem & Lösningar

*(Denna sektion fylls i när vi stöter på problem)*

---

## Kompilering & Körning

### Server:
```bash
cd Lab1_HTTP-Server/server
go build -o http_server main.go
./http_server 8080
```

### Proxy:
```bash
cd Lab1_HTTP-Server/proxy
go build -o proxy main.go
./proxy 9000
```

---

## Resurser

- [RFC 1945 - HTTP/1.0](https://tools.ietf.org/html/rfc1945)
- [Go net package](https://pkg.go.dev/net)
- [Go net/http package](https://pkg.go.dev/net/http)
- Lab README: `Lab1_HTTP-Server/README.md`

---

## Sammanfattning av nuvarande status

### ✅ Vad som fungerar (Session 1-3, 2025-11-11):
- TCP socket server på dynamisk port (command-line argument)
- Command-line port validation och error handling
- GET requests med filhantering
- POST requests med file upload
- Content-Type headers för alla filändelser
- Validering av file extensions (whitelist)
- Validering av HTTP methods (GET/POST)
- HTTP status codes: 200, 400, 404, 501
- Manual HTTP parsing (lärande-fokuserad approach)

### ❌ Vad som saknas för att klara labben:
1. **Concurrency control** - Max 10 samtidiga connections (KRITISKT KRAV)
2. **Proxy implementation** - Värd 7 poäng av totalt 17

### 📊 Uppskattad tid till färdig lab:
- Concurrency control: 2-3h
- Proxy implementation: 8-10h
- Testing & debugging: 2-3h
- **Total tid kvar:** ~12-16h fokuserad arbete

### 💡 Lärdomar hittills:
- Incremental development med testing fungerar utmärkt
- Manual parsing ger djup förståelse av HTTP-protokollet
- Go's error handling är konsekvent och tydlig
- Helper functions gör koden mer läsbar
- Testing efter varje steg förhindrar stora buggar
- Command-line argument parsing är enklare än förväntat
- `os.Exit()` är bättre än `panic()` för user errors

---

**Senast uppdaterad:** 2025-11-11 (Session 3)
**Nästa session:** Implementera concurrency control (goroutines + semaphore)
