# Multi-stage Dockerfile for QUIC Reverse Proxy

# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -X main.Version=docker' \
    -o quic-proxy \
    ./cmd/proxy

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates openssl curl docker && \
    addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S -G appgroup appuser

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/quic-proxy .

# Copy configuration files
COPY --from=builder /app/configs ./configs

# Create directories for certificates and logs
RUN mkdir -p certs logs && \
    chown -R appuser:appgroup /app

# NOTE: Running as root for docker socket access (control API)
# USER appuser

# Expose ports
# QUIC/HTTP3 proxy port
EXPOSE 443/udp
# Metrics port
EXPOSE 9090/tcp
# Health check port  
EXPOSE 8080/tcp

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Default command
ENTRYPOINT ["./quic-proxy"]
CMD ["-config", "configs/proxy.yaml"]