# Test Script - Generate Traffic Through Proxy

Write-Host "🚀 QUIC Proxy Metrics Test Generator" -ForegroundColor Cyan
Write-Host "=====================================" -ForegroundColor Cyan
Write-Host ""

# Configuration
$backendUrl = "http://localhost:8080"  # Direct backend (bypass proxy for now)
$proxyMetrics = "http://localhost:9090/metrics"
$requestCount = 50

Write-Host "📊 Current Setup:" -ForegroundColor Yellow
Write-Host "  Backend URL: $backendUrl"
Write-Host "  Metrics URL: $proxyMetrics"
Write-Host "  Test Requests: $requestCount"
Write-Host ""

# Generate traffic
Write-Host "🔄 Generating $requestCount requests..." -ForegroundColor Green

$successCount = 0
$errorCount = 0

for ($i = 1; $i -le $requestCount; $i++) {
    try {
        $response = Invoke-WebRequest -Uri $backendUrl -UseBasicParsing -Method GET -TimeoutSec 5
        if ($response.StatusCode -eq 200) {
            $successCount++
        }
    } catch {
        $errorCount++
    }
    
    # Progress indicator
    if ($i % 10 -eq 0) {
        Write-Host "  Progress: $i/$requestCount requests completed" -ForegroundColor Gray
    }
    
    Start-Sleep -Milliseconds 100
}

Write-Host ""
Write-Host "✅ Traffic Generation Complete!" -ForegroundColor Green
Write-Host "   Success: $successCount requests" -ForegroundColor Green
Write-Host "   Errors: $errorCount requests" -ForegroundColor Red
Write-Host ""

# Check metrics
Write-Host "📈 Fetching metrics..." -ForegroundColor Cyan

try {
    $metricsContent = (Invoke-WebRequest -Uri $proxyMetrics -UseBasicParsing).Content
    
    # Look for custom metrics
    $httpRequests = $metricsContent | Select-String "http_requests_total" | Measure-Object | Select-Object -ExpandProperty Count
    $backendRequests = $metricsContent | Select-String "backend_requests_total" | Measure-Object | Select-Object -ExpandProperty Count
    $goroutines = $metricsContent | Select-String "go_goroutines \d+" | Select-Object -First 1
    
    Write-Host ""
    Write-Host "📊 Metrics Status:" -ForegroundColor Yellow
    
    if ($httpRequests -gt 0) {
        Write-Host "  ✅ http_requests_total: FOUND ($httpRequests lines)" -ForegroundColor Green
    } else {
        Write-Host "  ❌ http_requests_total: NOT FOUND" -ForegroundColor Red
        Write-Host "     (Custom metrics not being recorded yet)" -ForegroundColor Gray
    }
    
    if ($backendRequests -gt 0) {
        Write-Host "  ✅ backend_requests_total: FOUND ($backendRequests lines)" -ForegroundColor Green
    } else {
        Write-Host "  ❌ backend_requests_total: NOT FOUND" -ForegroundColor Red
    }
    
    if ($goroutines) {
        Write-Host "  ✅ go_goroutines: $($goroutines.Line)" -ForegroundColor Green
    }
    
    Write-Host ""
    Write-Host "🔗 View all metrics: $proxyMetrics" -ForegroundColor Cyan
    Write-Host "📊 Grafana Dashboard: http://localhost:3001" -ForegroundColor Cyan
    Write-Host "🔍 Prometheus UI: http://localhost:9091" -ForegroundColor Cyan
    
} catch {
    Write-Host "❌ Failed to fetch metrics: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""
Write-Host "⚠️  NOTE: Currently testing direct backend access" -ForegroundColor Yellow
Write-Host "   The QUIC proxy listens on UDP port 443 (requires HTTP/3 client)" -ForegroundColor Yellow
Write-Host "   Metrics will show once we add HTTP fallback support" -ForegroundColor Yellow
Write-Host ""
