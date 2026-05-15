# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git make gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev
ARG GIT_COMMIT=unknown

LDFLAGS="-s -w -X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT}"

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="${LDFLAGS}" \
    -o api-server \
    ./cmd/api-server

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="${LDFLAGS}" \
    -o scheduler \
    ./cmd/scheduler

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="${LDFLAGS}" \
    -o node-agent \
    ./cmd/node-agent

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="${LDFLAGS}" \
    -o edge-cli \
    ./cmd/cli

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="${LDFLAGS}" \
    -o energy-market \
    ./cmd/energy-market

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="${LDFLAGS}" \
    -o agent-sandbox \
    ./cmd/agent-sandbox

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="${LDFLAGS}" \
    -o iot-connector \
    ./cmd/iot-connector

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="${LDFLAGS}" \
    -o coordination-engine \
    ./cmd/coordination-engine

FROM alpine:3.19 AS base-runtime

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -g '' appuser

# API Server runtime
FROM base-runtime AS api-server

WORKDIR /app

COPY --from=builder /build/api-server .
COPY config/ /app/config/

USER appuser

EXPOSE 8080 9090

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["./api-server"]

# Scheduler runtime
FROM base-runtime AS scheduler

WORKDIR /app

COPY --from=builder /build/scheduler .
COPY config/ /app/config/

USER appuser

EXPOSE 9090

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9090/health || exit 1

ENTRYPOINT ["./scheduler"]

# Node Agent runtime
FROM base-runtime AS node-agent

WORKDIR /app

COPY --from=builder /build/node-agent .
COPY config/ /app/config/

USER appuser

EXPOSE 9100

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9100/health || exit 1

ENTRYPOINT ["./node-agent"]

# CLI runtime
FROM base-runtime AS cli

WORKDIR /app

COPY --from=builder /build/edge-cli .

USER appuser

ENTRYPOINT ["./edge-cli"]

# Energy Market runtime
FROM base-runtime AS energy-market

WORKDIR /app

COPY --from=builder /build/energy-market .
COPY config/ /app/config/

RUN mkdir -p /data && chown appuser:appuser /data

USER appuser

EXPOSE 8081 9091

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

ENTRYPOINT ["./energy-market"]

# Agent Sandbox runtime
FROM base-runtime AS agent-sandbox

WORKDIR /app

RUN apk add --no-cache iptables iproute2

COPY --from=builder /build/agent-sandbox .
COPY config/ /app/config/

RUN mkdir -p /var/lib/edgehub/sandbox /var/lib/edgehub/storage && \
    chown -R appuser:appuser /var/lib/edgehub

EXPOSE 8082 9092

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8082/health || exit 1

ENTRYPOINT ["./agent-sandbox"]

# IoT Connector runtime
FROM base-runtime AS iot-connector

WORKDIR /app

COPY --from=builder /build/iot-connector .
COPY config/ /app/config/

USER appuser

EXPOSE 8083 9093

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8083/health || exit 1

ENTRYPOINT ["./iot-connector"]

# Coordination Engine runtime
FROM base-runtime AS coordination-engine

WORKDIR /app

COPY --from=builder /build/coordination-engine .
COPY config/ /app/config/

USER appuser

EXPOSE 8084 9094

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8084/health || exit 1

ENTRYPOINT ["./coordination-engine"]

# All-in-one image for development
FROM base-runtime AS all-in-one

WORKDIR /app

COPY --from=builder /build/api-server .
COPY --from=builder /build/scheduler .
COPY --from=builder /build/node-agent .
COPY --from=builder /build/energy-market .
COPY --from=builder /build/agent-sandbox .
COPY --from=builder /build/iot-connector .
COPY --from=builder /build/coordination-engine .
COPY config/ /app/config/

USER appuser

EXPOSE 8080 8081 8082 8083 8084 9090 9091 9092 9093 9094 9100

ENTRYPOINT ["./api-server"]
