# QUIC Reverse Proxy - Quick Reference Card

## 🚀 One-Minute Start

```powershell
# Windows PowerShell
.\test-proxy.ps1
```

That's it! The script handles everything automatically.

## 📝 Manual Setup (3 Steps)

### Step 1: Build
```bash
go build -o build/quic-proxy.exe ./cmd/proxy
```

### Step 2: Certificates
```bash
mkdir certs
openssl req -x509 -newkey rsa:2048 -keyout certs/server.key -out certs/server.crt -days 365 -nodes -subj "/CN=localhost"
```

### Step 3: Run
```bash
# Start backend
python -m http.server 8080

# Start proxy
.\build\quic-proxy.exe -config configs\proxy.yaml
```

## 🔧 Essential Commands

| Task | Command |
|------|---------|
| Build | `go build -o build/quic-proxy.exe ./cmd/proxy` |
| Run | `.\build\quic-proxy.exe -config configs\proxy.yaml` |
| Debug | `.\build\quic-proxy.exe -config configs\proxy.yaml -debug` |
| Version | `.\build\quic-proxy.exe -version` |
| Test | `go test ./...` |
| Clean | `Remove-Item -Recurse build\` |

## 🐳 Docker Commands

| Task | Command |
|------|---------|
| Build image | `docker build -t quic-proxy -f deployments/docker/Dockerfile .` |
| Start stack | `docker-compose up -d` |
| View logs | `docker-compose logs -f quic-proxy` |
| Stop stack | `docker-compose down` |
| Restart | `docker-compose restart quic-proxy` |

## 📊 Access Points

| Service | URL | Description |
|---------|-----|-------------|
| Proxy | `https://localhost:443` | Main QUIC proxy endpoint |
| Metrics | `http://localhost:9090/metrics` | Prometheus metrics |
| Grafana | `http://localhost:3001` | Visualization (admin/admin) |
| Jaeger | `http://localhost:16686` | Distributed tracing UI |
| Prometheus | `http://localhost:9091` | Metrics server UI |

## ⚙️ Configuration Locations

| File | Purpose |
|------|---------|
| `configs/proxy.yaml` | Main configuration |
| `configs/example.yaml` | Full options reference |
| `certs/server.{crt,key}` | TLS certificates |
| `monitoring/prometheus.yml` | Prometheus config |
| `docker-compose.yml` | Full stack orchestration |

## 📈 Key Metrics

```bash
# View all metrics
curl http://localhost:9090/metrics

# Key metrics to watch:
# - quic_connections_active
# - http_requests_total
# - http_request_duration_seconds
# - backend_health_status
```

## 🔍 Troubleshooting Quick Fixes

### Binary won't start
```bash
# Check configuration
.\build\quic-proxy.exe -config configs\proxy.yaml -debug

# Verify files exist
Test-Path certs\server.crt
Test-Path certs\server.key
Test-Path configs\proxy.yaml
```

### Can't connect to backends
```bash
# Test backend directly
curl http://localhost:8080

# Check logs with debug
.\build\quic-proxy.exe -config configs\proxy.yaml -debug
```

### Metrics not working
```yaml
# Verify in configs/proxy.yaml:
telemetry:
  metrics:
    enabled: true
    port: 9090
```

## 📁 Project Structure

```
quic-reverse-proxy/
├── build/              # Compiled binaries
│   └── quic-proxy.exe
├── configs/            # Configuration files
│   └── proxy.yaml
├── certs/              # TLS certificates
│   ├── server.crt
│   └── server.key
├── cmd/proxy/          # Main application
├── internal/           # Core implementation
│   ├── config/
│   ├── quic/
│   ├── proxy/
│   └── telemetry/
├── pkg/                # Shared packages
│   └── health/
└── deployments/        # Docker & K8s
    ├── docker/
    └── k8s/
```

## 🎯 Load Balancer Options

```yaml
backends:
  - name: "my-service"
    load_balancer: "round_robin"      # Fair distribution
    # OR
    load_balancer: "least_connections" # Dynamic routing
    # OR
    load_balancer: "weighted"          # Capacity-based
    weight: 5
```

## 🏥 Health Check Settings

```yaml
health_check:
  enabled: true
  path: "/health"              # Endpoint to check
  interval: "10s"              # How often
  timeout: "5s"                # Max wait time
  healthy_threshold: 2         # Successes to mark healthy
  unhealthy_threshold: 3       # Failures to mark unhealthy
```

## 🔐 QUIC Settings

```yaml
quic:
  max_streams: 1000            # Concurrent streams
  idle_timeout: "30s"          # Connection timeout
  keep_alive: "15s"            # Keep-alive interval
  enable_0rtt: true            # Fast reconnection
  congestion_algorithm: "cubic" # or "bbr", "reno"
```

## 📊 Logging Levels

```yaml
telemetry:
  logging:
    level: "debug"   # Verbose output
    # OR
    level: "info"    # Normal operation
    # OR
    level: "warn"    # Warnings only
    # OR
    level: "error"   # Errors only
```

## 🧪 Testing Checklist

- [ ] Binary builds: `go build -o build/quic-proxy.exe ./cmd/proxy`
- [ ] Certificates generated: `Test-Path certs\server.crt`
- [ ] Configuration valid: `Get-Content configs\proxy.yaml`
- [ ] Backend running: `curl http://localhost:8080`
- [ ] Proxy starts: `.\build\quic-proxy.exe -config configs\proxy.yaml`
- [ ] Metrics available: `curl http://localhost:9090/metrics`

## 📚 Documentation Files

| File | Content |
|------|---------|
| `README_COMPLETE.md` | Full user guide (start here!) |
| `PROJECT_SUMMARY.md` | Implementation overview |
| `COMPLETION_REPORT.md` | Build status & quick start |
| `docs/api.md` | API reference |
| This file | Quick reference card |

## 💡 Common Use Cases

### 1. Simple HTTP/3 Gateway
```yaml
backends:
  - name: "web"
    targets: ["http://localhost:8080"]
    load_balancer: "round_robin"
```

### 2. Load-Balanced API
```yaml
backends:
  - name: "api"
    targets:
      - "http://api1:8080"
      - "http://api2:8080"
      - "http://api3:8080"
    load_balancer: "least_connections"
    health_check:
      enabled: true
```

### 3. Weighted Distribution
```yaml
backends:
  - name: "prod"
    targets: ["http://prod-server:8080"]
    weight: 9
  - name: "canary"
    targets: ["http://canary-server:8080"]
    weight: 1
    load_balancer: "weighted"
```

## ⚡ Performance Tips

1. **Enable 0-RTT** for lower latency (repeat connections)
2. **Tune max_streams** based on your workload
3. **Use least_connections** for variable request durations
4. **Set appropriate timeouts** (not too short, not too long)
5. **Monitor metrics** to identify bottlenecks
6. **Use BBR congestion control** for better throughput

## 🎓 Learning Path

1. **Start**: Run `.\test-proxy.ps1` to see it in action
2. **Explore**: Read `README_COMPLETE.md`
3. **Configure**: Edit `configs/proxy.yaml` for your needs
4. **Deploy**: Use Docker Compose for full stack
5. **Monitor**: Check metrics and traces
6. **Optimize**: Tune based on your workload

## 🆘 Get Help

1. Check `README_COMPLETE.md` for detailed guides
2. Review `PROJECT_SUMMARY.md` for architecture
3. Read `COMPLETION_REPORT.md` for troubleshooting
4. Examine `configs/example.yaml` for all options
5. Enable debug mode: `-debug` flag

## ✨ Key Features at a Glance

- ✅ HTTP/3 (QUIC protocol)
- ✅ TLS 1.3 encryption
- ✅ 3 load balancing algorithms
- ✅ Advanced health checking
- ✅ Prometheus metrics
- ✅ OpenTelemetry tracing
- ✅ Structured logging
- ✅ Docker & Kubernetes ready
- ✅ Single binary deployment
- ✅ Graceful shutdown

---

**Status**: ✅ Ready to Use  
**Version**: 1.0.0  
**Last Updated**: October 11, 2025

**Quick Links**:
- Full Guide: `README_COMPLETE.md`
- Implementation: `PROJECT_SUMMARY.md`
- Quick Start: `COMPLETION_REPORT.md`