# stage: 0
FROM golang:1.17 AS builder

WORKDIR /go/src/workspace

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Add application code and install binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -a -v \
    -tags netgo \
    -o build/ingester \
    /go/src/workspace/cmd/ingester

# stage: 1
FROM alpine:latest

COPY --from=builder \
    /go/src/workspace/build/ingester \
    /usr/local/bin/

ENTRYPOINT [ "ingester" ]
