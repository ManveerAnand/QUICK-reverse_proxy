# QUIC Reverse Proxy

A high-performance reverse proxy built with QUIC/HTTP-3 support, featuring comprehensive telemetry, load balancing, and health monitoring.

## Features

🚀 **Core Functionality**
- QUIC/HTTP-3 reverse proxy with TLS 1.3
- Multiple load balancing algorithms (round-robin, least-connections, weighted)
- Advanced health checking with configurable thresholds
- Graceful shutdown and connection handling

📊 **Observability & Monitoring**
- Prometheus metrics collection
- OpenTelemetry distributed tracing with Jaeger
- Structured logging with configurable levels
- Real-time connection and request metrics
- Backend health status tracking

🔧 **Production-Ready**
- Comprehensive configuration management
- TLS certificate validation and generation
- Docker containerization
- Kubernetes deployment manifests
- CI/CD pipeline integration

## Quick Start

### Prerequisites

- Go 1.21 or later
- OpenSSL (for certificate generation)
- Docker and Docker Compose (for containerized deployment)

### Local Development

1. **Initialize the project:**
   ```bash
   make init-project
   ```

2. **Start backend services:**
   ```bash
   # Terminal 1: Node.js backend
   cd examples/node-backend
   npm install
   npm start

   # Terminal 2: Another backend (or use Docker)
   python -m http.server 8080
   ```

3. **Run the proxy:**
   ```bash
   make run
   ```

4. **Test the proxy:**
   ```bash
   # HTTP/3 request (requires curl with HTTP/3 support)
   curl --http3 https://localhost:443/api/status -k

   # Check metrics
   curl http://localhost:9090/metrics

   # Health check
   curl http://localhost:8080/health
   ```

### Docker Deployment

1. **Start the full stack:**
   ```bash
   make docker-compose-up
   ```

2. **Access services:**
   - QUIC Proxy: `https://localhost:443` 
   - Prometheus: `http://localhost:9091`
   - Grafana: `http://localhost:3001` (admin/admin)
   - Jaeger UI: `http://localhost:16686`

## Configuration

The proxy is configured via YAML files. See `configs/proxy.yaml` for the main configuration:

```yaml
server:
  address: ":443"
  tls:
    cert_file: "certs/server.crt"
    key_file: "certs/server.key"

backends:
  - name: "backend1"
    url: "http://localhost:8080"
    weight: 1
    
load_balancer:
  algorithm: "round_robin"  # round_robin, least_connections, weighted
  
health_check:
  enabled: true
  interval: "30s"
  timeout: "5s"
  path: "/health"

telemetry:
  metrics:
    enabled: true
    address: ":9090"
  tracing:
    enabled: true
    jaeger_endpoint: "http://localhost:14268/api/traces"
```

## Architecture

![QUIC Reverse Proxy Architecture](arch_os.png)

### Overview

The QUIC Reverse Proxy architecture consists of multiple layers working together to provide high-performance, secure HTTP/3 connectivity:

```
Client (HTTP/3) → QUIC Proxy → Load Balancer → Backend Services
                      ↓
                 Telemetry Stack
                 ├── Prometheus (Metrics)
                 ├── Jaeger (Tracing)  
                 └── Structured Logs
```

### Key Components

- **QUIC Server**: Handles HTTP/3 connections with TLS 1.3
- **Load Balancer**: Distributes requests across backends
- **Health Checker**: Monitors backend service health
- **Telemetry Manager**: Coordinates metrics and tracing
- **Configuration Manager**: Handles YAML configuration loading

## Development

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Build with debugging
make dev
```

### Testing

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage

# Run benchmarks
make benchmark

# Run linting
make lint
```

## Deployment

### Docker

```bash
# Build image
make docker-build

# Run container
make docker-run
```

### Kubernetes

```bash
# Deploy to cluster
kubectl apply -f deployments/k8s/
```

## License

This project is licensed under the MIT License. See the LICENSE file for details.