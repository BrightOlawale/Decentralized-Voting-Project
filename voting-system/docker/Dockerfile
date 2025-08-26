# syntax=docker/dockerfile:1

# -------- Build --------
FROM golang:1.23-bookworm AS builder
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential ca-certificates && \
    update-ca-certificates && rm -rf /var/lib/apt/lists/*
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# CGO is required for sqlite; keep it enabled
RUN CGO_ENABLED=1 GOOS=linux GOARCH=$(go env GOARCH) go build -o server ./cmd/server/main.go

# -------- Runtime --------
FROM debian:bookworm-slim
WORKDIR /app
RUN useradd -u 10001 -m appuser && mkdir -p /app/logs && chown -R appuser:appuser /app && \
    apt-get update && apt-get install -y --no-install-recommends ca-certificates && \
    update-ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/server /app/server
COPY configs /app/configs
COPY webapp /app/webapp
USER appuser
EXPOSE 8080
ENV SERVER_HOST=0.0.0.0 SERVER_PORT=8080
ENTRYPOINT ["/app/server"] 