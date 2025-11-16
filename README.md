## command to build docker image
at root folder 

build server(linux):
```docker build -f server/Dockerfile --tag docker-server .```

build server(mac):
```docker buildx build -f server/Dockerfile --platform linux/amd64,linux/arm64 -t daryl1104/docker-server:latest --push .```

build proxy(linux):
```docker build -f proxy/Dockerfile --tag docker-proxy .```

build proxy(mac):
```docker buildx build -f proxy/Dockerfile --platform linux/amd64,linux/arm64 -t daryl1104/docker-proxy:latest --push .```

run server locally:
```docker run -d --rm -p 8080:8080 --name myserver docker-server```

run proxy locally:
```docker run -d --rm -p 8081:8081 --name myproxy docker-proxy```

for pushing server to docker hub:
```docker tag docker-server daryl1104/docker-server:lastest```
```docker push daryl1104/docker-server:latest```

for pushing proxy to docker hub:
```docker tag docker-proxy daryl1104/docker-proxy:lastest```
```docker push daryl1104/docker-proxy:latest```

following step in cloud:
pull and run.

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

## login to aws
```chmod 400 "loginkey.pem"```
```ssh -i "loginkey.pem" ubuntu@ec2-44-223-37-23.compute-1.amazonaws.com```