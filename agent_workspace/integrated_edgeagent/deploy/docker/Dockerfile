FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o edgeagent-hub ./cmd/edgeagent

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/edgeagent-hub .
COPY --from=builder /app/config ./config

EXPOSE 8080

CMD ["./edgeagent-hub"]