FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -ldflags="-s -w" -o server ./cmd/mcp-file-tools

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/server ./

ENTRYPOINT ["./server"]
