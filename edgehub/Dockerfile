# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git make

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w" \
    -o api-server \
    ./cmd/api-server

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w" \
    -o scheduler \
    ./cmd/scheduler

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w" \
    -o node-agent \
    ./cmd/node-agent

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w" \
    -o edge-cli \
    ./cmd/cli

# API Server runtime
FROM alpine:3.19 AS api-server

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/api-server .
COPY config/ /app/config/

RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8080 9090

ENTRYPOINT ["./api-server"]

# Scheduler runtime
FROM alpine:3.19 AS scheduler

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/scheduler .
COPY config/ /app/config/

RUN adduser -D -g '' appuser
USER appuser

EXPOSE 9090

ENTRYPOINT ["./scheduler"]

# Node Agent runtime
FROM alpine:3.19 AS node-agent

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/node-agent .
COPY config/ /app/config/

RUN adduser -D -g '' appuser
USER appuser

EXPOSE 9100

ENTRYPOINT ["./node-agent"]

# CLI runtime
FROM alpine:3.19 AS cli

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /build/edge-cli .

RUN adduser -D -g '' appuser
USER appuser

ENTRYPOINT ["./edge-cli"]
