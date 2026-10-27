# QUIC Reverse Proxy Control Panel Guide

## Overview
The Control Panel provides a web-based interface to manage backend servers, monitor health status, and test failover scenarios in real-time.

## Access
- **Control Panel**: http://localhost:8889/control
- **Grafana Dashboard**: http://localhost:3001
- **Prometheus Metrics**: http://localhost:9090

## Features

### 1. Backend Cards
Each backend displays:
- **Health Indicator**: Green (healthy) / Red (unhealthy)
- **Backend Name**: Clean container name (backend1, backend2)
- **Status**: URL and health status
- **Metrics**: Active connections and load balancing weight

### 2. Control Buttons
- **🛑 Stop**: Stops the backend container
  - Enabled when backend is **healthy**
  - Triggers automatic failover to other backends
- **▶️ Start**: Starts the backend container
  - Enabled when backend is **unhealthy**
  - Backend rejoins load balancer after health checks pass
- **🔄 Restart**: Restarts the backend container
  - Always enabled
  - Useful for applying configuration changes

### 3. Simulation Tools
- **Generate Load**: Sends 50 test requests to measure distribution
- **Simulate Crash**: Randomly stops a healthy backend
- **Simulate Recovery**: Randomly starts an unhealthy backend
- **Clear Logs**: Clears the activity log display

### 4. Live Activity Logs
- Real-time log of all actions and events
- Color-coded by severity (info, warning, error)
- Auto-scrolls to show latest activity

### 5. Monitoring Links
Quick access to:
- Grafana dashboards for metrics visualization
- Prometheus for raw metrics data

## How Failover Works

### Automatic Health Checking
- **Interval**: Every 3 seconds
- **Timeout**: 2 seconds per check
- **Unhealthy Threshold**: 2 consecutive failures (6 seconds total)
- **Healthy Threshold**: 2 consecutive successes (6 seconds total)

### Failover Process
1. **Backend Goes Down**
   - Health check fails after 6 seconds (2 × 3s interval)
   - Backend marked as unhealthy
   - Load balancer removes backend from rotation
   - All new traffic routes to healthy backends only

2. **Traffic Rerouting**
   - Automatic and immediate after health status change
   - No manual intervention required
   - Zero downtime for end users

3. **Backend Recovery**
   - Start backend via Control Panel or Docker CLI
   - Health checks pass after 6 seconds (2 × 3s interval)
   - Backend automatically rejoins load balancer
   - Traffic distribution resumes across all healthy backends

### Testing Failover

#### Manual Test
1. Open Control Panel: http://localhost:8889/control
2. Note initial traffic distribution (both backends serving requests)
3. Click **🛑 Stop** on backend1
4. Wait 6 seconds for health check to detect failure
5. Observe:
   - Start button becomes **enabled** (green)
   - Stop button becomes **disabled** (greyed out)
   - Health indicator turns **red**
   - All traffic goes to backend2 (check Grafana)
6. Click **▶️ Start** on backend1
7. Wait 6 seconds for health check to detect recovery
8. Observe:
   - Start button becomes **disabled** (greyed out)
   - Stop button becomes **enabled**
   - Health indicator turns **green**
   - Traffic distribution resumes across both backends

#### Automated Test
Run the comprehensive test script:
```powershell
.\test_failover.ps1
```

This script:
- Checks initial health status
- Tests traffic distribution
- Stops backend1 via Control API
- Verifies automatic failover
- Starts backend1 via Control API
- Verifies automatic recovery
- Validates load balancing restored

## Architecture

### Components
```
┌─────────────────────────────────────────────────────────┐
│              Control Panel (Port 8889)                   │
│  - Web UI for backend management                        │
│  - Docker container control via API                     │
│  - Real-time status monitoring                          │
└──────────────────┬──────────────────────────────────────┘
                   │
┌──────────────────▼──────────────────────────────────────┐
│         QUIC Reverse Proxy (Ports 443, 80)              │
│  - QUIC/HTTP3 and HTTPS termination                     │
│  - Round-robin load balancing                           │
│  - Health checking (3s interval)                        │
│  - Automatic failover                                   │
└──────────┬────────────────────┬─────────────────────────┘
           │                    │
    ┌──────▼──────┐      ┌─────▼──────┐
    │  Backend1   │      │  Backend2  │
    │  (Nginx)    │      │  (Node.js) │
    │  Port: 80   │      │  Port: 3000│
    └─────────────┘      └────────────┘
```

### Health Check Flow
```
Proxy → GET http://backend1:80/ → 200 OK → Mark Healthy
Proxy → GET http://backend1:80/ → Timeout/Error → Increment Failure Count
Proxy → 2 consecutive failures → Mark Unhealthy → Remove from rotation
Proxy → 2 consecutive successes → Mark Healthy → Add to rotation
```

### Control API Flow
```
UI → POST /api/backend/stop?name=backend1 → Docker CLI: docker stop backend1
UI → POST /api/backend/start?name=backend1 → Docker CLI: docker start backend1
UI → POST /api/backend/restart?name=backend1 → Docker CLI: docker restart backend1
```

## Configuration

### Health Check Settings
Located in `configs/proxy.yaml`:
```yaml
health_check:
  enabled: true
  path: "/"              # Health check endpoint
  interval: "3s"         # Check every 3 seconds
  timeout: "2s"          # 2 second timeout
  healthy_threshold: 2   # 2 successes to mark healthy
  unhealthy_threshold: 2 # 2 failures to mark unhealthy
```

### UI Auto-Refresh
Located in `www/control.html`:
```javascript
setInterval(fetchStatus, 2000); // Refresh every 2 seconds
```

## Troubleshooting

### Issue: Start button stays greyed out
**Cause**: Health check hasn't detected failure yet  
**Solution**: Wait 6 seconds for 2 consecutive health check failures

### Issue: Traffic not rerouting
**Cause**: Health checks disabled or interval too long  
**Solution**: Verify `health_check.enabled: true` in proxy.yaml

### Issue: Backend shows unhealthy but container is running
**Cause**: Health check endpoint returning non-200 status  
**Solution**: 
- Check backend logs: `docker logs backend1`
- Verify health endpoint accessible: `docker exec backend1 curl http://localhost:80/`

### Issue: Control API not working
**Cause**: Docker socket not mounted or permission issues  
**Solution**:
- Verify `/var/run/docker.sock` mounted in docker-compose.yml
- Check container runs as root (not appuser)
- Verify docker CLI installed in container

## Best Practices

1. **Always test failover** before deploying to production
2. **Monitor Grafana** during failover to verify traffic routing
3. **Set appropriate health check intervals** (3-5s recommended)
4. **Use unhealthy threshold ≥ 2** to avoid false positives
5. **Enable health checks** for all production backends
6. **Keep logs** for troubleshooting failover issues

## Metrics to Monitor

### Grafana Dashboards
- **Backend Request Rate**: Requests per second per backend
- **Backend Health Status**: Real-time health indicator
- **Response Times**: p50, p95, p99 latencies
- **Error Rates**: 4xx and 5xx responses
- **Active Connections**: Current connections per backend

### Key Indicators
- **Healthy Backends Count**: Should match total backends
- **Request Distribution**: Should be roughly equal in round-robin
- **Failover Time**: Time from failure to traffic reroute (~6s)
- **Recovery Time**: Time from start to healthy status (~6s)

## Summary

The Control Panel provides:
✅ **Real-time monitoring** of backend health  
✅ **Manual control** over backend lifecycle  
✅ **Automatic failover** with zero downtime  
✅ **Fast detection** of failures (6 seconds)  
✅ **Visual feedback** via color-coded status  
✅ **Comprehensive testing** tools for validation  

For production use, integrate with:
- Alerting (Prometheus Alertmanager)
- Logging (ELK stack)
- Auto-scaling (Kubernetes HPA)
- Distributed tracing (Jaeger)
