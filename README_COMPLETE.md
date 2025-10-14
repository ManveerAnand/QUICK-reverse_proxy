# QUIC Reverse Proxy

A high-performance HTTP/3 (QUIC) reverse proxy with advanced telemetry, load balancing, and health checking capabilities.

## Features

✨ **Core Features**
- HTTP/3 (QUIC protocol) support for modern web applications
- Multiple load balancing algorithms (Round Robin, Least Connections, Weighted)
- Health checking with configurable thresholds
- TLS 1.3 support with automatic certificate management
- Graceful shutdown with connection draining

📊 **Telemetry & Monitoring**
- Prometheus metrics for connection, request, and backend statistics
- OpenTelemetry distributed tracing with Jaeger integration
- Structured logging with multiple output formats
- Real-time metrics dashboard support

🔧 **Configuration**
- YAML-based configuration
- Hot reload support (planned)
- Environment variable overrides
- Comprehensive validation with defaults

## Quick Start

### Prerequisites

- Go 1.21 or later
- OpenSSL (for certificate generation)
- Make (optional, for convenience commands)

### Installation

1. **Clone the repository**
```bash
git clone https://github.com/os-dev/quic-reverse-proxy.git
cd quic-reverse-proxy
```

2. **Install dependencies**
```bash
go mod download
```

3. **Generate test certificates**
```bash
mkdir certs
openssl req -x509 -newkey rsa:2048 -keyout certs/server.key -out certs/server.crt -days 365 -nodes \
    -subj "/C=US/ST=CA/L=SF/O=Dev/CN=localhost" \
    -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
```

4. **Build the proxy**
```bash
go build -o build/quic-proxy.exe ./cmd/proxy
```

### Running the Proxy

1. **Update configuration** (edit `configs/proxy.yaml`):
```yaml
server:
  address: "0.0.0.0:443"
  cert_file: "certs/server.crt"
  key_file: "certs/server.key"
  quic:
    max_streams: 100
    idle_timeout: 30s
    keep_alive: 10s
    enable_0rtt: false

backends:
  - name: "backend1"
    targets:
      - "localhost:8080"
      - "localhost:8081"
    load_balancer: "round_robin"
    health_check:
      enabled: true
      path: "/health"
      interval: 10s
      timeout: 5s
```

2. **Start backend services** (for testing):
```bash
# Terminal 1 - Simple HTTP server on port 8080
python -m http.server 8080

# Terminal 2 - Node.js backend (if you have it)
cd examples/node-backend
npm install
npm start
```

3. **Run the proxy**:
```bash
./build/quic-proxy.exe -config configs/proxy.yaml -debug
```

4. **Test the proxy**:
```bash
# Using curl with HTTP/3 support
curl --http3 https://localhost:443/ --insecure

# Or using a modern browser that supports HTTP/3
# Visit: https://localhost:443
```

## Configuration Reference

### Server Configuration

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `address` | string | Listen address for QUIC server | `0.0.0.0:443` |
| `cert_file` | string | Path to TLS certificate | Required |
| `key_file` | string | Path to TLS private key | Required |

### QUIC Settings

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `max_streams` | int | Maximum concurrent streams | `100` |
| `idle_timeout` | duration | Connection idle timeout | `30s` |
| `keep_alive` | duration | Keep-alive interval | `10s` |
| `enable_0rtt` | bool | Enable 0-RTT connection | `false` |

### Backend Configuration

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `name` | string | Backend identifier | Required |
| `targets` | []string | List of backend addresses | Required |
| `load_balancer` | string | Algorithm: `round_robin`, `least_connections`, `weighted` | `round_robin` |
| `weight` | int | Weight for weighted LB | `1` |
| `timeout` | duration | Backend request timeout | `30s` |

### Health Check Configuration

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `enabled` | bool | Enable health checks | `true` |
| `path` | string | Health check endpoint path | `/health` |
| `interval` | duration | Check interval | `10s` |
| `timeout` | duration | Check timeout | `5s` |
| `healthy_threshold` | int | Consecutive successes needed | `2` |
| `unhealthy_threshold` | int | Consecutive failures to mark unhealthy | `3` |

## Monitoring

### Prometheus Metrics

The proxy exposes metrics on port `9090` (configurable):

```bash
curl http://localhost:9090/metrics
```

**Key Metrics:**
- `quic_connections_total` - Total QUIC connections
- `quic_connections_active` - Active connections
- `http_requests_total` - Total HTTP requests
- `http_request_duration_seconds` - Request latency histogram
- `backend_health_status` - Backend health (1=healthy, 0=unhealthy)
- `backend_requests_total` - Requests per backend
- `quic_handshake_duration_seconds` - QUIC handshake latency

### Jaeger Tracing

Configure tracing in `telemetry.tracing`:

```yaml
telemetry:
  tracing:
    enabled: true
    endpoint: "http://localhost:14268/api/traces"
    service_name: "quic-proxy"
    sample_rate: 0.1
```

Access Jaeger UI: `http://localhost:16686`

## Development

### Build Commands

```bash
# Build for current platform
go build -o build/quic-proxy ./cmd/proxy

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o build/quic-proxy-linux ./cmd/proxy

# Build with debug symbols
go build -gcflags="all=-N -l" -o build/quic-proxy-debug ./cmd/proxy
```

### Run Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with race detection
go test -race ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Development Mode (Hot Reload)

Install Air:
```bash
go install github.com/cosmtrek/air@latest
```

Run with hot reload:
```bash
air
```

## Docker Deployment

### Build Docker Image

```bash
docker build -t quic-reverse-proxy:latest -f deployments/docker/Dockerfile .
```

### Run with Docker Compose

```bash
# Start all services (proxy, backends, monitoring)
docker-compose up -d

# View logs
docker-compose logs -f quic-proxy

# Stop all services
docker-compose down
```

The Docker Compose setup includes:
- QUIC Reverse Proxy
- Example backend services
- Prometheus for metrics
- Grafana for visualization
- Jaeger for tracing
- Redis for caching (optional)

## Kubernetes Deployment

```bash
# Deploy to Kubernetes
kubectl apply -f deployments/k8s/

# Check status
kubectl get pods -l app=quic-proxy

# View logs
kubectl logs -l app=quic-proxy -f
```

## Troubleshooting

### Common Issues

**1. Connection Refused**
- Ensure backend services are running
- Check firewall rules for UDP port 443
- Verify certificate paths in configuration

**2. Certificate Errors**
- Regenerate certificates with correct SANs
- Ensure certificate files are readable
- Check certificate expiration

**3. High Latency**
- Tune QUIC parameters (`idle_timeout`, `max_streams`)
- Check backend health and capacity
- Review load balancer algorithm selection

**4. Metrics Not Working**
- Verify metrics port is not blocked
- Check telemetry configuration
- Ensure Prometheus can reach the metrics endpoint

### Debug Mode

Run with debug logging:
```bash
./build/quic-proxy.exe -config configs/proxy.yaml -debug
```

Enable verbose logging in config:
```yaml
telemetry:
  logging:
    level: "debug"
    format: "json"
```

## Performance Tuning

### QUIC Optimization

```yaml
quic:
  max_streams: 200              # Increase for high concurrency
  idle_timeout: 60s             # Longer timeout for persistent connections
  keep_alive: 15s               # Balance between responsiveness and overhead
  enable_0rtt: true             # Enable for lower latency (security trade-off)
  congestion_algorithm: "bbr"   # Use BBR for better throughput
```

### Load Balancing

- **Round Robin**: Best for evenly distributed workloads
- **Least Connections**: Best for variable request durations
- **Weighted**: Use when backends have different capacities

### Health Checks

```yaml
health_check:
  interval: 5s                  # Faster detection of failures
  timeout: 2s                   # Quick timeout for faster failover
  healthy_threshold: 1          # Faster recovery
  unhealthy_threshold: 2        # Quick failure detection
```

## Architecture

```
┌─────────────┐
│   Clients   │
└──────┬──────┘
       │ HTTP/3 (QUIC)
       ▼
┌─────────────────────────────────────┐
│       QUIC Reverse Proxy            │
│  ┌──────────┐  ┌──────────────┐   │
│  │  QUIC    │  │  Load         │   │
│  │  Server  │─▶│  Balancer     │   │
│  └──────────┘  └──────┬───────┘   │
│                        │            │
│  ┌──────────────────────┴───────┐  │
│  │   Telemetry & Metrics        │  │
│  └──────────────────────────────┘  │
└───────────┬─────────────────────────┘
            │
    ┌───────┴───────┐
    ▼               ▼
┌─────────┐   ┌─────────┐
│Backend 1│   │Backend 2│
└─────────┘   └─────────┘
```

## Contributing

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Submit a pull request

## License

MIT License - see LICENSE file for details

## Resources

- [QUIC Protocol (RFC 9000)](https://www.rfc-editor.org/rfc/rfc9000.html)
- [HTTP/3 (RFC 9114)](https://www.rfc-editor.org/rfc/rfc9114.html)
- [quic-go Documentation](https://github.com/quic-go/quic-go)
- [Prometheus Metrics](https://prometheus.io/docs/)
- [OpenTelemetry](https://opentelemetry.io/)

## Support

For issues, questions, or contributions:
- GitHub Issues: [Report a bug or request a feature]
- Documentation: See `docs/` directory
- Examples: See `examples/` directory

---

**Built with ❤️ using Go and QUIC**