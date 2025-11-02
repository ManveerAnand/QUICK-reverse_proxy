# QUIC Reverse Proxy

A high-performance reverse proxy built with QUIC/HTTP-3 support, featuring comprehensive telemetry, load balancing, and health monitoring.

## Features

🚀 **Core Functionality**
- QUIC/HTTP-3 reverse proxy with TLS 1.3
- HTTP/1.1 fallback support for testing and compatibility
- Multiple load balancing algorithms (round-robin, least-connections, weighted)
- Advanced health checking with configurable thresholds
- Graceful shutdown and connection handling
- Path-based routing with wildcard support

📊 **Observability & Monitoring**
- Comprehensive Prometheus metrics (HTTP requests, latency, backend health)
- Pre-built Grafana dashboards with 10+ performance panels
- OpenTelemetry distributed tracing with Jaeger
- Structured logging with configurable levels
- Real-time connection and request metrics
- Backend health status tracking
- Custom metrics: request rates, latency percentiles (p50/p95/p99), success rates

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

   # HTTP/1.1 fallback (for testing)
   curl http://localhost:80/

   # Check metrics
   curl http://localhost:9090/metrics

   # Health check
   curl http://localhost:8888/health
   ```

## Health Checking and Failover

The QUIC reverse proxy includes a robust health checking system that ensures high availability and automatic failover.

### Health Check Configuration

```yaml
health_check:
  enabled: true
  interval: 10s           # Check every 10 seconds
  timeout: 5s            # Request timeout
  path: /health          # Health check endpoint
  unhealthy_threshold: 3 # Mark unhealthy after 3 failures
  healthy_threshold: 2   # Mark healthy after 2 successes
```

### How Failover Works

1. **Continuous Monitoring**
   - Every backend is checked at regular intervals (default: 10 seconds)
   - HTTP GET request sent to configured health endpoint (`/health`)
   - Timeout enforced (default: 5 seconds)

2. **Failure Detection**
   - If a backend fails to respond or returns error (non-2xx status)
   - Failure counter increments
   - After reaching unhealthy threshold (default: 3 consecutive failures):
     - Backend marked as **unhealthy** ❌
     - Automatically removed from load balancer rotation
     - No traffic routed to failed backend

3. **Automatic Recovery**
   - Health checks continue for unhealthy backends
   - On successful health check, success counter increments
   - After reaching healthy threshold (default: 2 consecutive successes):
     - Backend marked as **healthy** ✅
     - Automatically re-added to load balancer rotation
     - Traffic resumes to recovered backend

4. **Real-time Status Monitoring**
   - Query `/health` endpoint for current status:
     ```bash
     curl http://localhost:80/health
     ```
   - Response includes:
     ```json
     {
       "status": "healthy",          // Overall: healthy/degraded/unhealthy
       "healthy_backends": 2,        // Number of healthy backends
       "total_backends": 3,          // Total configured backends
       "timestamp": "2025-11-02T12:00:00Z"
     }
     ```
   - Status values:
     - `healthy` - All backends operational ✅
     - `degraded` - Some backends down (partial capacity) ⚠️
     - `unhealthy` - All backends failed ❌

### Control Center for Failover Testing

Access the web-based control panel to simulate and manage failover scenarios:

```bash
# Open control center
http://localhost:8889/control
```

**Control Panel Features:**
- 🎮 **Real-time Backend Status** - Live health indicators and stats
- 🛑 **Stop Backend** - Manually stop containers to test failover
- ▶️ **Start Backend** - Restart failed backends
- 🔄 **Restart Backend** - Graceful restart with connection draining
- 📊 **Grafana Integration** - Direct links to performance dashboards
- 🔥 **Load Generation** - Simulate traffic (50 req/s)
- 💥 **Crash Simulation** - Randomly stop healthy backend
- 📝 **Activity Logs** - Real-time event tracking

**API Endpoints:**
```bash
# Stop a backend
curl -X POST http://localhost:8889/api/backend/stop?name=backend1

# Start a backend
curl -X POST http://localhost:8889/api/backend/start?name=backend1

# Restart a backend
curl -X POST http://localhost:8889/api/backend/restart?name=backend1
```

### Failover Scenario Example

**Scenario:** Backend1 crashes during production traffic

1. **T+0s** - Backend1 becomes unresponsive (simulated crash)
2. **T+10s** - First health check fails
3. **T+20s** - Second health check fails
4. **T+30s** - Third health check fails → Backend1 marked unhealthy
5. **T+30s** - Load balancer removes backend1 from rotation
6. **T+30s** - All traffic automatically routed to backend2 and backend3
7. **Status:** System status changes from `healthy` to `degraded`

**Recovery:**
1. **T+60s** - Administrator restarts backend1 via control panel
2. **T+70s** - First health check succeeds
3. **T+80s** - Second health check succeeds → Backend1 marked healthy
4. **T+80s** - Load balancer re-adds backend1 to rotation
5. **T+80s** - Traffic resumes to backend1 in round-robin fashion
6. **Status:** System status returns to `healthy`

### Monitoring Failover Events

**Prometheus Metrics:**
```bash
# Check backend health status
backend_health{backend="backend1"} = 1  # 1=healthy, 0=unhealthy

# Monitor health check failures
health_check_failures_total{backend="backend1"}

# Track backend uptime
backend_uptime_seconds{backend="backend1"}
```

**Grafana Dashboards:**
- Navigate to: http://localhost:3001/d/quic-proxy-performance
- Panels show:
  - Backend distribution (traffic shift during failover)
  - Request latency (impact of backend loss)
  - Success rate (should remain 100% during graceful failover)

**Jaeger Tracing:**
- View request flows: http://localhost:16686
- Trace failed requests to identify issues
- Monitor backend selection in load balancer

**Log Monitoring:**
```bash
# Watch proxy logs for health check events
docker logs -f quic-reverse-proxy | grep -i health

# Sample log output:
# {"level":"warn","backend":"backend1","msg":"Health check failed"}
# {"level":"info","backend":"backend1","msg":"Marked backend as unhealthy"}
# {"level":"info","backend":"backend1","msg":"Removed from load balancer"}
# {"level":"info","backend":"backend1","msg":"Health check passed"}
# {"level":"info","backend":"backend1","msg":"Marked backend as healthy"}
# {"level":"info","backend":"backend1","msg":"Added to load balancer"}
```

### Best Practices

1. **Tune Thresholds Based on Traffic**
   - High-traffic: Lower interval (5s), stricter thresholds (2 failures)
   - Low-traffic: Higher interval (15s), lenient thresholds (5 failures)

2. **Health Endpoint Design**
   - Should be lightweight (< 100ms response time)
   - Check critical dependencies (database, cache, external APIs)
   - Return detailed status info in response body

3. **Graceful Degradation**
   - Configure minimum healthy backends
   - Enable circuit breakers for cascading failures
   - Set appropriate timeout values

4. **Testing Failover**
   - Use control panel to simulate failures
   - Monitor Grafana dashboards during tests
   - Verify zero request drops during failover
   - Test recovery process thoroughly

### Docker Deployment

1. **Start the full stack:**
   ```bash
   make docker-compose-up
   ```

2. **Access services:**
   - QUIC Proxy (HTTP/3): `https://localhost:443` 
   - HTTP Fallback: `http://localhost:80`
   - **Control Center**: `http://localhost:8889/control` 🎮
   - Proxy Metrics: `http://localhost:9090/metrics`
   - Proxy Health: `http://localhost:80/health`
   - Prometheus: `http://localhost:9091`
   - Grafana: `http://localhost:3001` (admin/admin)
   - Jaeger UI: `http://localhost:16686`

3. **Configure Grafana (First-time setup):**
   ```bash
   # Access Grafana at http://localhost:3001
   # Login: admin / admin (change password when prompted)
   
   # Import the performance dashboard:
   # 1. Go to Dashboards → Import
   # 2. Upload: monitoring/grafana/dashboards/quic-proxy-dashboard.json
   # 3. Select "Prometheus" as the datasource
   # 4. Click "Import"
   ```

4. **Generate test traffic:**
   ```powershell
   # Windows PowerShell
   1..50 | ForEach-Object { curl http://localhost/; Start-Sleep -Milliseconds 100 }
   ```
   
   ```bash
   # Linux/Mac
   for i in {1..50}; do curl http://localhost/; sleep 0.1; done
   ```

## Configuration

The proxy is configured via YAML files. See `configs/proxy.yaml` for the main configuration:

```yaml
server:
  address: ":443"
  fallback_address: ":80"  # HTTP/1.1 fallback for testing
  tls:
    cert_file: "certs/server.crt"
    key_file: "certs/server.key"

backends:
  - name: "backend1"
    url: "http://backend1:80"
    weight: 1
  - name: "backend2"
    url: "http://backend2:3000"
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
    jaeger_endpoint: "http://jaeger:14268/api/traces"
  logging:
    level: "info"
    format: "json"
```

### Monitoring Configuration

The proxy exposes comprehensive metrics at `:9090/metrics`:

**Custom Metrics:**
- `http_requests_total` - Total HTTP requests (by method, status_code, backend)
- `http_request_duration_seconds` - Request latency histogram
- `http_request_size_bytes` - Request payload sizes
- `http_response_size_bytes` - Response payload sizes
- `backend_requests_total` - Backend request counts (by backend, status)
- `backend_response_time_seconds` - Backend latency histogram

**Go Runtime Metrics:**
- `go_goroutines` - Active goroutines
- `go_memstats_alloc_bytes` - Memory allocation
- `process_cpu_seconds_total` - CPU usage

See `monitoring/grafana/dashboards/METRICS_AVAILABLE.md` for complete metric documentation.

## Architecture

```
Client (HTTP/3) → QUIC Proxy (Port 443) → Load Balancer → Backend Services
Client (HTTP/1) → HTTP Server (Port 80) ↗                  ↓
                                                    Health Checker
                      ↓
                 Telemetry Stack
                 ├── Prometheus (Port 9091) - Metrics Collection
                 ├── Grafana (Port 3001) - Visualization
                 ├── Jaeger (Port 16686) - Distributed Tracing
                 └── Structured Logs
```

### Components

- **QUIC Server**: Handles HTTP/3 connections with TLS 1.3 on port 443
- **HTTP Fallback Server**: HTTP/1.1 server on port 80 for testing and compatibility
- **Load Balancer**: Distributes requests across backends using configurable algorithms
- **Health Checker**: Monitors backend service health with periodic checks
- **Telemetry Manager**: Coordinates metrics collection and distributed tracing
- **Metrics Exporter**: Exposes Prometheus metrics at `:9090/metrics`
- **Configuration Manager**: Handles YAML configuration loading and validation

## Grafana Dashboard

The project includes a pre-built Grafana dashboard with 10 performance panels:

### Dashboard Panels

1. **HTTP Request Rate** - Requests/sec by method and backend
2. **Total Requests** - Cumulative request counter
3. **Success Rate %** - Percentage of successful requests (200 status)
4. **Request Latency (Percentiles)** - p50, p95, p99 latency tracking
5. **Backend Request Distribution** - Traffic distribution across backends
6. **Backend Response Time (p95)** - 95th percentile backend latency
7. **Request/Response Sizes** - Data transfer metrics
8. **Active Goroutines** - Concurrency monitoring
9. **Memory Usage** - Heap allocation in MB
10. **CPU Usage** - Process CPU utilization

### Importing the Dashboard

1. Access Grafana at `http://localhost:3001`
2. Login with `admin/admin` (change password on first login)
3. Navigate to **Dashboards** → **Import**
4. Click **Upload JSON file**
5. Select `monitoring/grafana/dashboards/quic-proxy-dashboard.json`
6. Choose **Prometheus** as the datasource
7. Click **Import**

The dashboard auto-refreshes every 10 seconds and shows the last 15 minutes of data.

### Available Dashboards

- `quic-proxy-dashboard.json` - Main performance dashboard with custom metrics
- Additional dashboards can be created using the metrics documented in `METRICS_AVAILABLE.md`

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