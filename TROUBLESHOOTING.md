# 🔧 Troubleshooting & FAQ Guide

> **Purpose**: Solutions to common problems, debugging techniques, and frequently asked questions about the QUIC Reverse Proxy.

---

## 📚 Table of Contents

1. [Common Errors](#common-errors)
2. [Build & Compilation Issues](#build--compilation-issues)
3. [Runtime Problems](#runtime-problems)
4. [Performance Issues](#performance-issues)
5. [Connection Problems](#connection-problems)
6. [Certificate & TLS Issues](#certificate--tls-issues)
7. [Backend Communication](#backend-communication)
8. [Monitoring & Logging](#monitoring--logging)
9. [FAQ](#faq)
10. [Advanced Debugging](#advanced-debugging)

---

## ⚠️ Common Errors

### Error: `bind: address already in use`

**Symptom**:
```
FATAL: Failed to start server: bind: address already in use
```

**Cause**: Another process is already listening on port 8443 (or your configured port).

**Solution 1: Find and stop the process**
```bash
# Find process using the port
lsof -i:8443  # Linux/Mac
netstat -ano | findstr :8443  # Windows

# Output shows:
# COMMAND   PID   USER
# proxy     1234  user

# Kill the process
kill 1234  # Linux/Mac
taskkill /PID 1234 /F  # Windows
```

**Solution 2: Change the port**
```yaml
# In configs/proxy.yaml
server:
  address: ":8444"  # Use different port
```

**Solution 3: Clean up zombie processes**
```bash
# Kill all proxy processes
pkill -9 proxy  # Linux/Mac
taskkill /IM proxy.exe /F  # Windows
```

---

### Error: `certificate verify failed`

**Symptom**:
```
curl: (60) SSL certificate problem: self signed certificate
```

**Cause**: Using self-signed certificate (development) that client doesn't trust.

**Solution 1: Skip verification (development only)**
```bash
# curl
curl --http3 -k https://localhost:8443

# wget
wget --no-check-certificate https://localhost:8443

# Browser: Click "Advanced" → "Proceed anyway"
```

**Solution 2: Add certificate to trust store**
```bash
# Linux
sudo cp certs/server.crt /usr/local/share/ca-certificates/
sudo update-ca-certificates

# Mac
sudo security add-trusted-cert -d -r trustRoot \
  -k /Library/Keychains/System.keychain certs/server.crt

# Windows
certutil -addstore "Root" certs/server.crt
```

**Solution 3: Use proper certificate (production)**
```bash
# Get Let's Encrypt certificate
certbot certonly --standalone -d your-domain.com

# Update config
server:
  cert_file: "/etc/letsencrypt/live/your-domain.com/fullchain.pem"
  key_file: "/etc/letsencrypt/live/your-domain.com/privkey.pem"
```

---

### Error: `no healthy backends available`

**Symptom**:
```
HTTP 503 Service Unavailable
{"error": "no healthy backends available"}
```

**Cause**: All backends in the group are marked unhealthy or none are running.

**Diagnosis**:
```bash
# Check if backends are running
lsof -i:8001
lsof -i:8002
lsof -i:8003

# Test backend directly
curl http://localhost:8001/health

# Check proxy logs
tail -f logs/proxy.log | grep "health_check"
```

**Solution 1: Start backends**
```bash
# Start your backend servers
cd examples/node-backend
npm start &

# Or manually
python3 -m http.server 8001 &
```

**Solution 2: Fix health check endpoint**
```yaml
# Ensure health check path matches backend
health_check:
  path: "/health"  # Backend must have this endpoint
```

**Solution 3: Disable health checks temporarily**
```yaml
# For debugging only
health_check:
  enabled: false  # All backends assumed healthy
```

**Solution 4: Check health check logs**
```bash
# Look for specific error messages
tail -f logs/proxy.log | jq 'select(.message == "health_check_failed")'

# Common errors:
# - "connection refused" → Backend not running
# - "timeout" → Backend too slow or network issue
# - "404 not found" → Wrong health check path
# - "500 internal error" → Backend has internal issue
```

---

### Error: `dial tcp: i/o timeout`

**Symptom**:
```
ERROR: Backend request failed: dial tcp 10.0.0.1:8001: i/o timeout
```

**Cause**: Backend is unreachable (network issue, firewall, or wrong address).

**Diagnosis**:
```bash
# Test network connectivity
ping 10.0.0.1

# Test port specifically
nc -zv 10.0.0.1 8001  # Linux/Mac
Test-NetConnection -ComputerName 10.0.0.1 -Port 8001  # PowerShell

# Check firewall
sudo iptables -L  # Linux
sudo ufw status  # Ubuntu
netsh advfirewall show allprofiles  # Windows
```

**Solution 1: Fix backend URL**
```yaml
# Check if URL is correct
backends:
  - url: "http://localhost:8001"  # ✅ Correct
  # not: "http://10.0.0.1:8001" if unreachable
```

**Solution 2: Allow firewall**
```bash
# Linux
sudo iptables -A INPUT -p tcp --dport 8001 -j ACCEPT

# Ubuntu
sudo ufw allow 8001/tcp

# Windows
netsh advfirewall firewall add rule name="Backend" dir=in action=allow protocol=TCP localport=8001
```

**Solution 3: Increase timeout**
```yaml
timeout:
  connect: 10s  # Increase from 5s
  request: 60s
```

---

## 🔨 Build & Compilation Issues

### Error: `go: missing go.sum entry`

**Symptom**:
```
go: missing go.sum entry for module providing package ...
```

**Solution**:
```bash
# Regenerate go.sum
go mod tidy

# Verify modules
go mod verify

# If still failing, clear cache
go clean -modcache
go mod download
```

---

### Error: `undefined: quic.Config`

**Symptom**:
```
internal/proxy/server.go:45:12: undefined: quic.Config
```

**Cause**: Dependency not downloaded or wrong version.

**Solution**:
```bash
# Install specific version
go get github.com/quic-go/quic-go@v0.40.0

# Update dependencies
go mod tidy

# Rebuild
go build -o build/proxy ./cmd/proxy
```

---

### Error: Build takes too long

**Symptom**: `go build` hangs or takes 5+ minutes.

**Diagnosis**:
```bash
# Check if downloading modules
go build -x -o build/proxy ./cmd/proxy
# Shows every command being executed
```

**Solution 1: Use module cache**
```bash
# Download modules first
go mod download

# Then build (much faster)
go build -o build/proxy ./cmd/proxy
```

**Solution 2: Use build cache**
```bash
# Check cache status
go env GOCACHE

# Clean if corrupted
go clean -cache

# Build with verbose output
go build -v -o build/proxy ./cmd/proxy
```

**Solution 3: Disable CGO (if not needed)**
```bash
# CGO can slow down builds
CGO_ENABLED=0 go build -o build/proxy ./cmd/proxy
```

---

## 🚀 Runtime Problems

### Proxy starts but doesn't accept connections

**Symptom**: Proxy logs show "started" but clients get "connection refused".

**Diagnosis**:
```bash
# Check if proxy is actually listening
lsof -i:8443  # Should show proxy process
netstat -tuln | grep 8443  # Should show LISTEN

# Check logs for errors
tail -f logs/proxy.log

# Test with telnet
telnet localhost 8443
```

**Solution 1: Check bind address**
```yaml
# Wrong (only listens on localhost)
server:
  address: "127.0.0.1:8443"

# Correct (listens on all interfaces)
server:
  address: "0.0.0.0:8443"
```

**Solution 2: Check firewall**
```bash
# Allow incoming connections
sudo ufw allow 8443/tcp  # Ubuntu
sudo iptables -A INPUT -p tcp --dport 8443 -j ACCEPT  # Linux
```

**Solution 3: Check SELinux (Linux)**
```bash
# Check if SELinux is blocking
sudo setenforce 0  # Temporarily disable

# If that fixes it, add permanent rule
sudo semanage port -a -t http_port_t -p tcp 8443
```

---

### Proxy crashes randomly

**Symptom**:
```
panic: runtime error: invalid memory address or nil pointer dereference
```

**Diagnosis**:
```bash
# Enable panic recovery and logging
export GOTRACEBACK=crash

# Run with race detector
go run -race ./cmd/proxy -config configs/proxy.yaml

# Check system resources
top  # CPU and memory usage
dmesg | tail  # Kernel messages (OOM killer?)
```

**Common causes**:

1. **Nil pointer dereference**
   - Backend group not found
   - Config field not set
   - Solution: Add nil checks in code

2. **Out of memory**
   ```bash
   # Check memory usage
   pmap <proxy-pid>
   
   # Reduce connection limits
   max_idle_connections: 50  # Reduce from 100
   ```

3. **Too many open files**
   ```bash
   # Check limit
   ulimit -n
   
   # Increase limit
   ulimit -n 65536
   
   # Make permanent (Linux)
   echo "* soft nofile 65536" >> /etc/security/limits.conf
   echo "* hard nofile 65536" >> /etc/security/limits.conf
   ```

4. **Goroutine leak**
   ```bash
   # Check number of goroutines
   curl http://localhost:9090/debug/pprof/goroutine
   
   # Profile goroutines
   go tool pprof http://localhost:9090/debug/pprof/goroutine
   ```

---

### High memory usage

**Symptom**: Proxy uses 1GB+ RAM after running for hours.

**Diagnosis**:
```bash
# Check memory profile
curl -o mem.prof http://localhost:9090/debug/pprof/heap

# Analyze profile
go tool pprof mem.prof
> top10  # Show top 10 memory consumers
> list <function>  # Show source code
```

**Common causes and solutions**:

1. **Connection pool too large**
   ```yaml
   connection_pool:
     max_idle_connections: 50  # Reduce
     idle_timeout: 60s  # Close idle faster
   ```

2. **Logs not rotating**
   ```yaml
   logging:
     output: "logs/proxy.log"
     max_size: 100  # MB
     max_backups: 3
     max_age: 7  # days
   ```

3. **Response body buffering**
   ```go
   // Don't do this (loads entire response in memory)
   body, _ := io.ReadAll(resp.Body)
   
   // Do this instead (streaming)
   io.Copy(w, resp.Body)
   ```

4. **Metrics cardinality explosion**
   ```yaml
   # Don't create metrics with unbounded labels
   # Bad: http_requests{path="/user/12345"}  # Unique per user!
   # Good: http_requests{path_pattern="/user/*"}
   ```

---

## ⚡ Performance Issues

### Low throughput (< 1000 req/s)

**Diagnosis**:
```bash
# CPU profiling
curl -o cpu.prof http://localhost:9090/debug/pprof/profile?seconds=30

# Analyze
go tool pprof cpu.prof
> top10
> web  # Visual graph (requires graphviz)
```

**Common bottlenecks**:

1. **Single-threaded processing**
   ```bash
   # Check CPU usage
   top -H -p <proxy-pid>
   
   # If only one core is maxed, increase GOMAXPROCS
   export GOMAXPROCS=8  # Use 8 CPU cores
   ```

2. **Backend is slow**
   ```bash
   # Check backend latency
   curl -o /dev/null -s -w "Time: %{time_total}s\n" http://localhost:8001/test
   
   # Check backend CPU/memory
   top -p <backend-pid>
   ```

3. **Connection pool exhausted**
   ```yaml
   connection_pool:
     max_connections_per_host: 50  # Increase
     max_idle_connections: 200
   ```

4. **Too much logging**
   ```yaml
   logging:
     level: "warn"  # Change from "debug"
   ```

---

### High latency (> 100ms)

**Diagnosis**:
```bash
# Check latency distribution
curl http://localhost:9090/metrics | grep duration_seconds

# Trace specific slow request
# (requires OpenTelemetry setup)
```

**Common causes**:

1. **Network latency to backend**
   ```bash
   # Measure network latency
   ping backend-host
   
   # Traceroute
   traceroute backend-host
   ```

2. **Health check contention**
   ```yaml
   health_check:
     interval: 30s  # Increase from 10s (less frequent checks)
   ```

3. **Mutex contention (lock争用)**
   ```bash
   # Check mutex profile
   curl -o mutex.prof http://localhost:9090/debug/pprof/mutex
   go tool pprof mutex.prof
   ```

4. **GC pauses**
   ```bash
   # Check GC stats
   GODEBUG=gctrace=1 ./build/proxy -config configs/proxy.yaml
   
   # Reduce GC pressure
   export GOGC=200  # Run GC less frequently (default: 100)
   ```

---

## 🔌 Connection Problems

### Error: `http3: connection closed`

**Symptom**: Random connection closures during transfers.

**Cause**: Idle timeout, keepalive timeout, or network issue.

**Solution 1: Increase timeouts**
```yaml
server:
  max_idle_timeout: 60s  # Increase from 30s
  keep_alive_period: 10s  # More frequent keepalives
```

**Solution 2: Enable keepalive on client**
```bash
# curl with keepalive
curl --http3 --keepalive-time 10 https://localhost:8443
```

**Solution 3: Check network stability**
```bash
# Test packet loss
ping -c 100 backend-host

# Check for high packet loss (> 1%)
```

---

### Connection migration fails

**Symptom**: Connection drops when switching networks (WiFi → Ethernet).

**Cause**: Connection ID not preserved or NAT rebinding issue.

**Solution 1: Ensure migration is enabled**
```yaml
server:
  allow_connection_migration: true  # Should be default
```

**Solution 2: Check NAT/firewall**
- Some corporate firewalls block UDP port changes
- Test on different network

**Solution 3: Use stateless reset**
```yaml
quic_config:
  enable_stateless_reset: true
```

---

## 🔐 Certificate & TLS Issues

### Error: `tls: failed to find any PEM data`

**Symptom**:
```
FATAL: Failed to load TLS certificate: tls: failed to find any PEM data
```

**Cause**: Certificate or key file is corrupted, wrong format, or empty.

**Diagnosis**:
```bash
# Check file contents
cat certs/server.crt
# Should start with: -----BEGIN CERTIFICATE-----

cat certs/server.key
# Should start with: -----BEGIN PRIVATE KEY-----

# Verify certificate
openssl x509 -in certs/server.crt -text -noout

# Verify key
openssl rsa -in certs/server.key -check
```

**Solution**: Regenerate certificates
```bash
rm certs/server.crt certs/server.key
make certs
```

---

### Error: `tls: private key does not match public key`

**Symptom**:
```
FATAL: tls: private key does not match public key in certificate
```

**Cause**: Certificate and key are from different key pairs.

**Solution**: Ensure cert and key are generated together
```bash
# Generate new matching pair
openssl req -x509 -newkey rsa:4096 -keyout certs/server.key \
  -out certs/server.crt -days 365 -nodes \
  -subj "/CN=localhost"

# Or use existing key to generate cert
openssl req -new -x509 -key certs/server.key -out certs/server.crt -days 365
```

---

### Error: `certificate has expired`

**Symptom**:
```
curl: (60) SSL certificate problem: certificate has expired
```

**Diagnosis**:
```bash
# Check certificate validity
openssl x509 -in certs/server.crt -noout -dates

# Output:
# notBefore=Oct 1 00:00:00 2024 GMT
# notAfter=Oct 1 00:00:00 2025 GMT  ← Expired!
```

**Solution**: Generate new certificate
```bash
# Remove old cert
rm certs/server.crt certs/server.key

# Generate new cert (valid 1 year)
make certs

# For production, renew Let's Encrypt
certbot renew
```

---

## 🔗 Backend Communication

### Backend receives duplicate requests

**Symptom**: Backend logs show same request multiple times.

**Cause**: Retry logic triggering on transient errors.

**Solution**: Adjust retry configuration
```yaml
retry:
  max_attempts: 1  # Disable retries
  # Or make retries more selective
  retry_on:
    - "connection_error"  # Only retry connection errors
    # Remove "5xx" to avoid retrying server errors
```

---

### Backend doesn't receive client headers

**Symptom**: Backend logs show missing `Authorization` header.

**Cause**: Header not being copied from client request.

**Diagnosis**:
```bash
# Check proxy logs
tail -f logs/proxy.log | jq '.headers'

# Test with verbose curl
curl --http3 -k -v -H "Authorization: Bearer token123" \
  https://localhost:8443/api/test
```

**Solution**: Ensure header forwarding is working
```go
// In proxy code, verify this exists:
for key, values := range r.Header {
    for _, value := range values {
        backendReq.Header.Add(key, value)
    }
}
```

---

### Backend sees wrong client IP

**Symptom**: Backend logs show proxy IP instead of real client IP.

**Cause**: `X-Forwarded-For` header not set or not used by backend.

**Solution 1: Check proxy adds header**
```yaml
routes:
  - id: "api"
    add_headers:
      X-Forwarded-For: "${client_ip}"
      X-Real-IP: "${client_ip}"
```

**Solution 2: Configure backend to trust proxy**
```javascript
// Express.js example
app.set('trust proxy', true);

app.get('/api/*', (req, res) => {
  const clientIP = req.ip;  // Now gets real IP from X-Forwarded-For
});
```

---

## 📊 Monitoring & Logging

### Metrics endpoint returns 404

**Symptom**:
```bash
curl http://localhost:9090/metrics
# 404 Not Found
```

**Diagnosis**:
```bash
# Check if metrics are enabled
cat configs/proxy.yaml | grep -A5 "telemetry"

# Check if metrics server is running
lsof -i:9090
```

**Solution**:
```yaml
telemetry:
  metrics:
    enabled: true
    port: 9090
    path: "/metrics"  # Ensure path is correct
```

---

### Logs are empty or missing

**Symptom**: `logs/proxy.log` is empty or doesn't exist.

**Diagnosis**:
```bash
# Check log configuration
cat configs/proxy.yaml | grep -A5 "logging"

# Check file permissions
ls -la logs/

# Check if directory exists
mkdir -p logs
```

**Solution 1: Fix output path**
```yaml
logging:
  output: "logs/proxy.log"  # Not "log/proxy.log"
```

**Solution 2: Use stdout for debugging**
```yaml
logging:
  output: "stdout"  # Print to terminal
```

**Solution 3: Increase log level**
```yaml
logging:
  level: "debug"  # More verbose
```

---

### Log file grows too large

**Symptom**: `logs/proxy.log` is 5GB+ and filling disk.

**Solution**: Enable log rotation
```yaml
logging:
  output: "logs/proxy.log"
  max_size: 100  # MB per file
  max_backups: 3  # Keep 3 old files
  max_age: 7  # Days to keep
  compress: true  # Compress old logs
```

**Manual rotation**:
```bash
# Rotate logs manually
mv logs/proxy.log logs/proxy.log.1
gzip logs/proxy.log.1

# Send SIGHUP to reload log file
kill -HUP <proxy-pid>
```

---

## ❓ FAQ

### Q: Why use QUIC instead of HTTP/2?

**A**: QUIC provides several advantages:

1. **Faster connection establishment**: 0-1 RTT vs 2-3 RTT for HTTP/2
2. **No head-of-line blocking**: Independent streams don't block each other
3. **Connection migration**: Seamless network changes (WiFi → Cellular)
4. **Better loss recovery**: Per-stream retransmission
5. **Built-in encryption**: TLS 1.3 mandatory

**When to use HTTP/2 instead**:
- Legacy client support needed
- UDP is blocked by firewall
- Running on older systems without QUIC support

---

### Q: Can I use this in production?

**A**: Yes, but with considerations:

**Production checklist**:
- [ ] Use proper TLS certificates (not self-signed)
- [ ] Enable monitoring (Prometheus + Grafana)
- [ ] Configure log rotation
- [ ] Set appropriate resource limits
- [ ] Test failover scenarios
- [ ] Load test with expected traffic
- [ ] Set up alerting
- [ ] Document incident response procedures

**Not recommended for**:
- Ultra-high traffic (> 50k req/s per instance) - scale horizontally instead
- Legacy clients requiring HTTP/1.1 only
- Environments where UDP is blocked

---

### Q: How many backends can I configure?

**A**: **Practical limit: ~100 backends per group**

**Reasoning**:
- Health checks scale linearly (100 backends = 100 checks per interval)
- Load balancer algorithms are O(n)
- Connection pool memory grows with backend count

**For more backends**:
- Split into multiple backend groups
- Use service mesh (Istio, Linkerd)
- Implement consistent hashing

---

### Q: What happens during backend deployment?

**Scenario**: Rolling update of backend servers.

**Without health checks**:
1. Backend 1 stops
2. Proxy sends requests → Connection refused
3. Clients see errors ❌

**With health checks** (recommended):
1. Backend 1 stops
2. Health check fails after ~30s (3× threshold)
3. Proxy stops routing to Backend 1
4. Clients continue working ✅
5. Backend 1 updates and restarts
6. Health check succeeds after ~20s (2× threshold)
7. Proxy resumes routing to Backend 1

**Best practice**: 
- Use gradual rollout (1 backend at a time)
- Wait for health checks between deployments

---

### Q: How to scale horizontally?

**Option 1: DNS round-robin**
```
Clients → DNS (myapp.com)
          ├→ proxy-1.myapp.com (Load 33%)
          ├→ proxy-2.myapp.com (Load 33%)
          └→ proxy-3.myapp.com (Load 34%)
```

**Option 2: Load balancer in front**
```
Clients → L4 Load Balancer
          ├→ Proxy 1
          ├→ Proxy 2
          └→ Proxy 3
           Each proxy → Backend pool
```

**Option 3: Kubernetes Service**
```yaml
apiVersion: v1
kind: Service
metadata:
  name: quic-proxy
spec:
  type: LoadBalancer
  selector:
    app: quic-proxy
  ports:
    - port: 443
      targetPort: 8443
```

---

### Q: Can I use this with WebSockets?

**A**: **Partial support**

- QUIC doesn't directly support WebSocket protocol
- Use HTTP/3 extended CONNECT method
- Or use QUIC streams as WebSocket alternative

**Alternative**: Use separate WebSocket proxy for now, or use HTTP/3 streaming

---

### Q: How to debug slow requests?

**Step-by-step debugging**:

1. **Enable distributed tracing**
   ```yaml
   telemetry:
     tracing:
       enabled: true
       endpoint: "localhost:4318"
   ```

2. **Send test request**
   ```bash
   curl --http3 -k https://localhost:8443/api/slow
   ```

3. **View trace in Jaeger** (http://localhost:16686)
   - Identify slowest span
   - Common slow components:
     - Backend processing (most common)
     - TLS handshake (first request)
     - Connection pool exhaustion

4. **Check metrics**
   ```bash
   curl http://localhost:9090/metrics | grep duration
   ```

5. **Analyze logs**
   ```bash
   cat logs/proxy.log | jq 'select(.duration_ms > 100)'
   ```

---

## 🔬 Advanced Debugging

### Enable pprof profiling

```go
// Add to cmd/proxy/main.go
import _ "net/http/pprof"

// Start pprof server
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

**Usage**:
```bash
# CPU profiling
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Memory profiling
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine profiling
go tool pprof http://localhost:6060/debug/pprof/goroutine

# Block profiling (mutex contention)
go tool pprof http://localhost:6060/debug/pprof/block
```

---

### Packet capture analysis

```bash
# Capture QUIC traffic
sudo tcpdump -i any -w quic.pcap 'udp port 8443'

# Analyze with Wireshark
wireshark quic.pcap

# Or use tshark
tshark -r quic.pcap -Y quic
```

---

### Stress testing

```bash
# Install hey (better than ab)
go install github.com/rakyll/hey@latest

# Run stress test
hey -n 100000 -c 1000 -q 10 https://localhost:8443/api/test

# Monitor during test
watch -n 1 'curl -s http://localhost:9090/metrics | grep http_requests'
```

---

## 📞 Getting Help

### Before asking for help

1. **Check logs** with debug level
   ```yaml
   logging:
     level: "debug"
   ```

2. **Review configuration** for typos
   ```bash
   cat configs/proxy.yaml | grep -i backend
   ```

3. **Test backends directly**
   ```bash
   curl http://localhost:8001/health
   ```

4. **Check system resources**
   ```bash
   top -H -p <proxy-pid>
   lsof -p <proxy-pid> | wc -l  # File descriptor count
   ```

5. **Review error message** carefully
   - Often contains solution in the message itself

---

### Creating bug reports

**Include**:
1. **Go version**: `go version`
2. **OS and version**: `uname -a`
3. **Proxy version/commit**: `git rev-parse HEAD`
4. **Configuration** (redacted): `cat configs/proxy.yaml`
5. **Error logs**: Last 50 lines with timestamps
6. **Steps to reproduce**: Exact commands to trigger issue
7. **Expected vs actual behavior**

**Example**:
```markdown
## Bug Report

### Environment
- Go: 1.21.3
- OS: Ubuntu 22.04
- Commit: abc123def456

### Configuration
```yaml
server:
  address: ":8443"
backends:
  - url: "http://localhost:8001"
```

### Steps to reproduce
1. Start proxy: `./build/proxy -config configs/proxy.yaml`
2. Send request: `curl --http3 -k https://localhost:8443/test`
3. Observe error in logs

### Expected behavior
Request succeeds with 200 OK

### Actual behavior
Error: "connection refused"

### Logs
```
[2025-10-14 10:30:00] ERROR: dial tcp: connection refused
```
```

---

**Created for**: Team support and self-service debugging
**Last Updated**: October 14, 2025
