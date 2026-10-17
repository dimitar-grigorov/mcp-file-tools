FROM golang:1.27-alpine@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o server ./cmd/mcp-file-tools

FROM alpine:latest@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6

WORKDIR /app
COPY --from=builder /app/server ./

ENTRYPOINT ["./server"]
