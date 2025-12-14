FROM golang:1.25

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY *.go ./
COPY private.pem cert.pem ./

RUN CGO_ENABLED=0 GOOS=linux go build -o ./chord

#EXPOSE 4170

ENTRYPOINT ["./chord"]
