# QUIC Reverse Proxy - Quick Test Script
# This script sets up a simple test environment

Write-Host "=== QUIC Reverse Proxy Quick Test ===" -ForegroundColor Cyan
Write-Host ""

# Check if binary exists
if (-not (Test-Path "build\quic-proxy.exe")) {
    Write-Host "[!] Binary not found. Building..." -ForegroundColor Yellow
    go build -o build\quic-proxy.exe .\cmd\proxy
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[X] Build failed!" -ForegroundColor Red
        exit 1
    }
    Write-Host "[OK] Build successful!" -ForegroundColor Green
}

# Check if certificates exist
if (-not (Test-Path "certs\server.crt")) {
    Write-Host "[*] Generating test certificates..." -ForegroundColor Yellow
    .\generate-certs.ps1
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[X] Failed to generate certificates" -ForegroundColor Red
        exit 1
    }
}

# Check configuration
if (-not (Test-Path "configs\proxy.yaml")) {
    Write-Host "[X] Configuration file not found: configs\proxy.yaml" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "=== Starting Simple Backend Server ===" -ForegroundColor Cyan

# Start a simple Python HTTP server as backend
$backendJob = Start-Job -ScriptBlock {
    Set-Location $using:PWD
    python -m http.server 8080 2>$null
}

Start-Sleep -Seconds 2

if ($backendJob.State -eq "Running") {
    Write-Host "[OK] Backend server started on http://localhost:8080" -ForegroundColor Green
} else {
    Write-Host "[X] Failed to start backend server. Is Python installed?" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "=== Configuration ===" -ForegroundColor Cyan
Write-Host "Proxy Address: https://localhost:443" -ForegroundColor White
Write-Host "Backend: http://localhost:8080" -ForegroundColor White
Write-Host "Metrics: http://localhost:9090/metrics" -ForegroundColor White
Write-Host "Certificates: certs/server.crt and certs/server.key" -ForegroundColor White

Write-Host ""
Write-Host "=== Starting QUIC Reverse Proxy ===" -ForegroundColor Cyan
Write-Host "Press Ctrl+C to stop" -ForegroundColor Gray
Write-Host ""

# Start the proxy
try {
    .\build\quic-proxy.exe -config configs\proxy.yaml -debug
} finally {
    Write-Host ""
    Write-Host "=== Cleaning up ===" -ForegroundColor Cyan
    Stop-Job -Job $backendJob
    Remove-Job -Job $backendJob
    Write-Host "[OK] Backend server stopped" -ForegroundColor Green
    Write-Host ""
    Write-Host "Test completed. Thank you!" -ForegroundColor Cyan
}
