# Generate Self-Signed Certificates for QUIC Proxy
# This script creates test certificates for local development

Write-Host "Generating self-signed certificates for QUIC Proxy..." -ForegroundColor Cyan

# Create certs directory
New-Item -ItemType Directory -Force -Path "certs" | Out-Null

# Create a temporary OpenSSL config file
$opensslConfig = @"
[req]
default_bits = 2048
prompt = no
default_md = sha256
distinguished_name = dn
x509_extensions = v3_req

[dn]
C = US
ST = California
L = San Francisco
O = Development
CN = localhost

[v3_req]
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = *.localhost
IP.1 = 127.0.0.1
IP.2 = ::1
"@

$configPath = "certs\openssl.cnf"
$opensslConfig | Out-File -FilePath $configPath -Encoding ASCII

Write-Host "Creating private key and certificate..." -ForegroundColor Yellow

# Generate certificate
$command = "openssl req -x509 -newkey rsa:2048 -keyout certs\server.key -out certs\server.crt -days 365 -nodes -config $configPath"

try {
    Invoke-Expression $command 2>$null
    
    if ((Test-Path "certs\server.crt") -and (Test-Path "certs\server.key")) {
        Write-Host "[OK] Certificate and key generated successfully!" -ForegroundColor Green
        Write-Host ""
        Write-Host "Certificate: certs\server.crt" -ForegroundColor White
        Write-Host "Private Key: certs\server.key" -ForegroundColor White
        Write-Host "Valid for: 365 days" -ForegroundColor White
        Write-Host ""
        Write-Host "Certificate Details:" -ForegroundColor Cyan
        openssl x509 -in certs\server.crt -noout -text | Select-String "Subject:|Not After|DNS:|IP Address:"
    } else {
        Write-Host "[X] Failed to generate certificates" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "[X] Error: $_" -ForegroundColor Red
    exit 1
} finally {
    # Clean up config file
    if (Test-Path $configPath) {
        Remove-Item $configPath -Force
    }
}

Write-Host ""
Write-Host "Certificates are ready to use!" -ForegroundColor Green
