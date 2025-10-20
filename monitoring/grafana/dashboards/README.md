# QUIC Reverse Proxy - Grafana Dashboard

## 📊 Dashboard Overview

This dashboard provides real-time monitoring of your QUIC reverse proxy using Go runtime metrics.

### Panels Included:

1. **Active Goroutines** - Monitor concurrent execution units
2. **Memory Usage (MB)** - Track memory allocation over time
3. **OS Threads** - Number of operating system threads
4. **Heap Objects** - Count of objects in memory heap
5. **GC System Memory** - Memory used by garbage collector
6. **Open File Descriptors** - System resource usage
7. **Garbage Collection Pause Time** - GC performance impact
8. **Metrics Scrape Rate** - Prometheus scraping frequency
9. **Memory Breakdown** - Detailed memory allocation (Heap/Stack)
10. **CPU Usage** - Process CPU consumption
11. **Garbage Collection Frequency** - GC runs per minute

---

## 🚀 How to Import Dashboard

### Method 1: Import via Grafana UI

1. **Open Grafana**: http://localhost:3001
2. **Login**: 
   - Username: `admin`
   - Password: `admin`
3. **Navigate**: Click **"Dashboards"** → **"Import"** (or click **"+"** → **"Import"**)
4. **Upload JSON**:
   - Click **"Upload JSON file"**
   - Select: `monitoring/grafana/dashboards/quic-proxy-dashboard.json`
5. **Configure Data Source**:
   - **Prometheus**: Select `Prometheus` from dropdown
6. **Import**: Click **"Import"** button

### Method 2: Copy-Paste JSON

1. Open Grafana → **Dashboards** → **Import**
2. Paste the contents of `quic-proxy-dashboard.json`
3. Click **"Load"**
4. Select **Prometheus** data source
5. Click **"Import"**

---

## ⚙️ Dashboard Settings

- **Refresh Interval**: 10 seconds (auto-refresh)
- **Time Range**: Last 15 minutes
- **Timezone**: Browser default
- **Tags**: `quic`, `proxy`, `golang`, `performance`

---

## 🔧 Before Importing - Configure Prometheus Data Source

If you haven't added Prometheus yet:

1. **Connections** → **Data Sources** → **Add data source**
2. Select **"Prometheus"**
3. Configure:
   ```
   Name: Prometheus
   URL: http://prometheus:9090
   ```
   ⚠️ **Important**: Use `prometheus:9090` (Docker network), NOT `localhost:9091`
4. Click **"Save & Test"** → Should see ✅ "Data source is working"

---

## 📈 Available Metrics

Current metrics exposed by the QUIC proxy:

### Go Runtime Metrics:
- `go_goroutines` - Active goroutines
- `go_memstats_*` - Memory statistics
- `go_gc_*` - Garbage collection metrics
- `go_threads` - OS threads
- `process_*` - Process-level metrics
- `promhttp_metric_handler_requests_total` - Scrape requests

### Custom QUIC Metrics (Coming Soon):
These are defined but not yet instrumented:
- `quic_connections_total` - QUIC connection count
- `quic_request_duration_seconds` - Request latency
- `http_requests_total` - HTTP request count
- `backend_health_status` - Backend health

---

## 🎨 Customization

### Change Refresh Rate:
1. Click the time picker (top right)
2. Change auto-refresh interval (5s, 10s, 30s, 1m, 5m)

### Modify Panels:
1. Click panel title → **"Edit"**
2. Modify query in **"Code"** tab
3. Adjust visualization in right panel
4. Click **"Apply"** → **"Save dashboard"**

### Add Alerts:
1. Edit panel → **"Alert"** tab
2. Create alert rule (e.g., "Memory > 100MB")
3. Configure notification channels

---

## 🐛 Troubleshooting

### "No Data" in Panels:
```bash
# Check Prometheus is scraping:
curl http://localhost:9091/targets

# Check metrics endpoint:
curl http://localhost:9090/metrics
```

### Connection Refused:
- Ensure all containers are running: `docker-compose ps`
- Restart Prometheus: `docker-compose restart prometheus`
- Check logs: `docker logs prometheus`

### Dashboard Import Failed:
- Verify JSON syntax is valid
- Ensure Prometheus data source exists
- Check Grafana version compatibility (10.0+)

---

## 📊 Dashboard UID

- **UID**: `quic-proxy-runtime`
- **Version**: 1.0
- **Created**: 2025-10-20

---

## 🔗 Quick Links

- **Grafana UI**: http://localhost:3001
- **Prometheus UI**: http://localhost:9091
- **Proxy Metrics**: http://localhost:9090/metrics
- **Jaeger Tracing**: http://localhost:16686

---

## 📝 Next Steps

1. ✅ Import this dashboard
2. 🔄 Make test requests to generate traffic
3. 📈 Watch metrics update in real-time (10s intervals)
4. 🎯 Instrument custom QUIC metrics in code
5. 🚀 Create alerts for anomalies

---

**Enjoy monitoring your QUIC proxy! 🚀**
