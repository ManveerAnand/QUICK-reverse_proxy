# QUIC Proxy Grafana Dashboard - Quick Import Guide

## 🎯 SUPER EASY IMPORT (3 Steps)

### Step 1: Open Import Page
```
http://localhost:3001/dashboard/import
```
Or: Grafana → Dashboards → New → Import

### Step 2: Upload File
Click **"Upload JSON file"** → Select:
```
monitoring/grafana/dashboards/quic-proxy-dashboard.json
```

### Step 3: Configure & Import
- **Prometheus**: Select `Prometheus` (or `prometheus`)
- Click **"Import"**

**DONE! 🎉**

---

## 📋 What You'll See:

### Top Row:
- 🔄 **Active Goroutines** (real-time concurrency)
- 💾 **Memory Usage** (MB allocated)

### Stats Cards:
- 🧵 OS Threads
- 📦 Heap Objects  
- 🗑️ GC Memory
- 📁 Open Files

### Bottom Graphs:
- ⏱️ GC Pause Time
- 🔍 Metrics Scrape Rate
- 📊 Memory Breakdown (stacked area)
- ⚡ CPU Usage
- 🔄 GC Frequency

---

## ⚡ Quick Test:

After importing:
1. Dashboard auto-refreshes **every 10 seconds**
2. All panels should show **green lines/numbers**
3. Zoom in/out with time picker (top right)
4. Hover over graphs to see values

---

## 🆘 If You See "No Data Source":

Run this first:
1. **Connections** → **Data Sources** → **Add data source**
2. Select **Prometheus**
3. URL: `http://prometheus:9090`
4. Click **Save & Test** ✅

Then import dashboard again!

---

**Dashboard UID**: `quic-proxy-runtime`
**File**: `monitoring/grafana/dashboards/quic-proxy-dashboard.json`
