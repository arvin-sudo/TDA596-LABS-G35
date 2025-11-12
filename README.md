# Important: don't use postman :( it
it reused tcp connections that goes into two seperate go routines even one request by clicking once.

## command to build docker image
at root folder 

for server build:
```docker build -f server/Dockerfile --tag docker-gs-ping . ```

for proxy build:
```docker build -f proxy/Dockerfile --tag docker-gs-ping-1 .```

for server run:
```docker run -d --rm -p 8080:8080 -e PORT=8080 --name myserver docker-gs-ping```

for proxy run:
```docker run -d --rm -p 8081:8081 -e PORT=8081 --name myproxy docker-gs-ping-1```

## test by using curl
server(get): 
```curl -i http://localhost:8082/files/test.txt```

server(post):
```curl -i -F "vv=@/Users/xinyi/Desktop/tmp.txt" http://localhost:8082/files/tmp.txt```

proxy:
```curl -X GET google.com -x localhost:8081```

## if you try to use nc to test, use this structure: 
```
POST /users.html HTTP/1.1
Host: example.com
Content-Type: application/x-www-form-urlencoded
Content-Length: 49

name=FirstName

```

