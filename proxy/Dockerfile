FROM golang:1.19

WORKDIR /app

COPY ../go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /docker-gs-ping

EXPOSE 8081

CMD ["/docker-gs-ping", "8081"]
