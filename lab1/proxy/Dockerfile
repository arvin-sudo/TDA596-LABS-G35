FROM golang:1.25

WORKDIR /app

COPY go.mod ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /docker-gs-ping ./proxy

EXPOSE 8081

CMD ["/docker-gs-ping", "8081"]
