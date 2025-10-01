FROM golang:1.24-alpine

WORKDIR /app

COPY node/go.mod node/go.sum ./
RUN go mod download

COPY node/ ./

RUN go build -o node-app main.go

CMD ["./node-app"]
