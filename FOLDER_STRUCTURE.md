# 📁 Complete Folder Structure & Code Explanation

> **Purpose**: This document walks through every file and directory in the project, explaining what it does, why it exists, and how it fits into the overall system.

---

## 📚 Table of Contents

1. [Project Root Files](#project-root-files)
2. [cmd/ - Application Entry Points](#cmd---application-entry-points)
3. [internal/ - Core Business Logic](#internal---core-business-logic)
4. [pkg/ - Reusable Libraries](#pkg---reusable-libraries)
5. [configs/ - Configuration Files](#configs---configuration-files)
6. [certs/ - TLS Certificates](#certs---tls-certificates)
7. [scripts/ - Automation Scripts](#scripts---automation-scripts)
8. [deployments/ - Deployment Configurations](#deployments---deployment-configurations)
9. [monitoring/ - Observability Setup](#monitoring---observability-setup)
10. [examples/ - Demo Applications](#examples---demo-applications)
11. [docs/ - Additional Documentation](#docs---additional-documentation)

---

## 📂 Project Root Files

### `go.mod` - Go Module Definition

**What it is**: This file defines your Go module and manages all dependencies.

**Why it exists**: Go uses modules (introduced in Go 1.11+) to manage dependencies. Instead of manually downloading libraries, Go reads this file to know what packages your project needs.

**Content explanation**:
```go
module github.com/ManveerAnand/quic-reverse-proxy

go 1.21  // Minimum Go version required

require (
    github.com/quic-go/quic-go v0.40.0  // QUIC protocol implementation
    github.com/prometheus/client_golang v1.17.0  // Metrics collection
    go.opentelemetry.io/otel v1.19.0  // Distributed tracing
    gopkg.in/yaml.v3 v3.0.1  // YAML parsing for config
    // ... more dependencies
)
```

**How it's used**: When you run `go build` or `go run`, Go automatically downloads all packages listed here.

---

### `go.sum` - Dependency Checksums

**What it is**: A lockfile containing cryptographic hashes of all dependencies.

**Why it exists**: Security and reproducibility. This file ensures that when someone else builds your project, they get the **exact same versions** of dependencies you used.

**Example content**:
```
github.com/quic-go/quic-go v0.40.0 h1:abc123def456...
github.com/quic-go/quic-go v0.40.0/go.mod h1:xyz789ghi012...
```

**How it works**: 
- When you `go get` a package, Go adds its hash to `go.sum`
- On subsequent builds, Go verifies the hash matches
- If hashes don't match → Security alert! Package may have been tampered with

**You should**: Commit this file to version control.

---

### `Makefile` - Build Automation

**What it is**: A script that defines shortcuts for common commands.

**Why it exists**: Instead of typing long commands like:
```bash
go build -o build/proxy ./cmd/proxy && ./build/proxy -config configs/proxy.yaml
```

You can just type:
```bash
make run
```

**Key targets explained**:

```makefile
# Build the proxy binary
build:
	@echo "Building QUIC Reverse Proxy..."
	@mkdir -p build
	go build -o build/proxy ./cmd/proxy

# Why: Creates executable in build/ directory
# The @ suppresses command echo, -o specifies output filename
```

```makefile
# Run the proxy with default config
run: build
	@echo "Starting proxy server..."
	./build/proxy -config configs/proxy.yaml

# Why: Builds first (dependency), then runs with config file
# This ensures you're always running the latest code
```

```makefile
# Run all tests with coverage
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Why: 
# -v: Verbose (shows each test)
# -race: Detect race conditions (concurrent access bugs)
# -coverprofile: Generate coverage report
# ./...: Test all packages recursively
```

```makefile
# Generate TLS certificates for development
certs:
	@echo "Generating self-signed certificates..."
	@mkdir -p certs
	openssl req -x509 -newkey rsa:4096 -keyout certs/server.key \
	  -out certs/server.crt -days 365 -nodes \
	  -subj "/CN=localhost"

# Why: QUIC requires TLS, this creates dev certificates
# -x509: Create self-signed cert
# -days 365: Valid for 1 year
# -nodes: No password protection (for dev)
```

```makefile
# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf build/ coverage.out coverage.html

# Why: Remove compiled binaries and test artifacts
# Useful when you want a fresh build
```

---

### `README.md` - Project Overview

**What it is**: The first thing people see when they visit your GitHub repository.

**What it contains**:
- Project description and purpose
- Quick start guide (how to build and run)
- Feature highlights
- Architecture diagram
- Links to detailed documentation

**Why it's important**: 
- GitHub displays this on your repo homepage
- Helps new contributors understand the project quickly
- Shows installation instructions

---

### `.gitignore` - Git Exclusion Rules

**What it is**: Tells Git which files to NOT track.

**Why it exists**: Some files shouldn't be in version control:
- Build artifacts (generated, not source)
- Sensitive data (certificates, passwords)
- IDE-specific files (personal preferences)
- Large temporary files

**Key sections explained**:

```gitignore
# Binaries - Don't track compiled executables
*.exe
*.dll
build/

# Why: These are generated from source code
# Other developers should build their own
```

```gitignore
# Go - Language-specific artifacts
vendor/   # Downloaded dependencies (use go.mod instead)
*.test    # Test binaries
*.out     # Test output files

# Why: These can be regenerated with 'go mod download' and 'go test'
```

```gitignore
# Certificates - Security sensitive
certs/*.crt
certs/*.key
!certs/.gitkeep  # But keep the directory structure

# Why: Real certificates contain private keys (security risk!)
# Only commit example/placeholder files
```

```gitignore
# IDE - Personal preferences
.vscode/
.idea/

# Why: Each developer uses different editors/settings
# Don't force your IDE config on others
```

```gitignore
# Logs - Too large and not useful in version control
*.log
logs/

# Why: Log files can grow to gigabytes
# They're runtime artifacts, not source code
```

---

## 📂 cmd/ - Application Entry Points

The `cmd/` directory contains **main packages** - these are the actual programs you can run.

### Why separate cmd/ from internal/?
- **cmd/**: "I want to run this as a program" (has `func main()`)
- **internal/**: "This is library code used by programs" (imported by cmd/)

This pattern is a Go convention for structuring projects.

---

### `cmd/proxy/` - The Main Proxy Application

This is the heart of the project - the actual reverse proxy server.

#### `cmd/proxy/main.go` - Entry Point

**What it does**: 
1. Parses command-line arguments
2. Loads configuration
3. Initializes all components
4. Starts the proxy server
5. Handles graceful shutdown

**Code walkthrough**:

```go
package main

import (
    "flag"
    "log"
    "os"
    "os/signal"
    "syscall"
)

func main() {
    // Step 1: Parse command-line flags
    configFile := flag.String("config", "configs/proxy.yaml", "Path to config file")
    flag.Parse()
```

**Explanation**: 
- `flag.String()` creates a command-line option: `-config configs/proxy.yaml`
- Users can override with: `./proxy -config /path/to/custom.yaml`
- `flag.Parse()` actually reads the command-line arguments

```go
    // Step 2: Load configuration from YAML file
    cfg, err := config.LoadConfig(*configFile)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
```

**Explanation**:
- Reads the YAML file specified by `-config` flag
- Parses it into a Go struct (defined in `internal/config/`)
- If parsing fails, exit immediately (can't run without config)

```go
    // Step 3: Initialize telemetry (metrics, tracing, logging)
    telemetryShutdown, err := telemetry.Initialize(cfg.Telemetry)
    if err != nil {
        log.Fatalf("Failed to initialize telemetry: %v", err)
    }
    defer telemetryShutdown()  // Ensure cleanup on exit
```

**Explanation**:
- Sets up Prometheus metrics endpoint
- Configures OpenTelemetry tracing
- Initializes structured logging
- `defer` ensures cleanup happens even if program crashes

```go
    // Step 4: Create backend manager (manages connection pools)
    backendManager := backend.NewManager(cfg.BackendGroups)
```

**Explanation**:
- Creates a manager to handle all backend server groups
- Each group has its own connection pool and health checker
- Manager handles load balancing between backends

```go
    // Step 5: Start health checks for all backend groups
    for _, group := range cfg.BackendGroups {
        if group.HealthCheck != nil && group.HealthCheck.Enabled {
            checker := health.NewChecker(group, backendManager)
            go checker.Start()  // Run in background goroutine
        }
    }
```

**Explanation**:
- Iterates through each backend group
- If health checks are enabled, create a checker
- `go checker.Start()` runs health checks in a separate goroutine (concurrent)
- Health checks run continuously in the background

```go
    // Step 6: Create and start the proxy server
    proxyServer, err := proxy.NewServer(cfg, backendManager)
    if err != nil {
        log.Fatalf("Failed to create proxy server: %v", err)
    }

    go func() {
        log.Printf("Starting QUIC proxy on %s", cfg.Server.Address)
        if err := proxyServer.ListenAndServe(); err != nil {
            log.Fatalf("Proxy server error: %v", err)
        }
    }()
```

**Explanation**:
- Creates the main QUIC server with all configuration
- Starts listening on specified address (e.g., `:443`)
- Runs in a goroutine so we can handle shutdown signals
- If server fails to start, entire application exits

```go
    // Step 7: Wait for shutdown signal (Ctrl+C)
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    <-sigChan  // Block until signal received

    log.Println("Shutdown signal received, gracefully stopping...")
```

**Explanation**:
- Creates a channel to receive OS signals
- `signal.Notify()` tells Go to send signals to this channel
- `<-sigChan` blocks (waits) until Ctrl+C is pressed
- This allows the server to run until you explicitly stop it

```go
    // Step 8: Graceful shutdown
    if err := proxyServer.Shutdown(); err != nil {
        log.Printf("Error during shutdown: %v", err)
    }
    
    backendManager.Close()  // Close all backend connections
    log.Println("Server stopped gracefully")
}
```

**Explanation**:
- `Shutdown()` stops accepting new connections
- Waits for existing requests to complete (graceful)
- Closes all backend connections
- Flushes telemetry data
- Clean exit without dropping active requests

**Why this structure?**
- **Separation of concerns**: Each step does one thing
- **Error handling**: Check errors immediately after each step
- **Graceful shutdown**: No dropped connections or data loss
- **Observability**: Telemetry initialized early to capture all events

---

## 📂 internal/ - Core Business Logic

The `internal/` directory contains all the core functionality. It's called "internal" because Go has a special rule: **packages in internal/ can only be imported by code in the same module**, not by external projects.

### Why use internal/?
- **Encapsulation**: Your internal code is not a public API
- **Freedom to change**: You can refactor without breaking external users
- **Clear boundaries**: Separates library code from application code

---

### `internal/config/` - Configuration Management

This package handles reading and validating configuration files.

#### `internal/config/config.go` - Configuration Structures

**What it does**: Defines the structure of the YAML configuration file.

**Code walkthrough**:

```go
package config

import (
    "fmt"
    "os"
    "time"
    "gopkg.in/yaml.v3"
)

// Config represents the entire configuration file
type Config struct {
    Server        ServerConfig       `yaml:"server"`
    Routes        []RouteConfig      `yaml:"routes"`
    BackendGroups []BackendGroup     `yaml:"backend_groups"`
    Telemetry     TelemetryConfig    `yaml:"telemetry"`
}
```

**Explanation**:
- This struct maps directly to the YAML file structure
- `yaml:"server"` tells the YAML parser which field to populate
- Each field is a nested struct (defined below)

**Example YAML**:
```yaml
server:
  address: ":443"
routes:
  - id: "api"
backend_groups:
  - id: "api-servers"
telemetry:
  metrics_enabled: true
```

Maps to:
```go
Config{
    Server: ServerConfig{Address: ":443"},
    Routes: []RouteConfig{{ID: "api"}},
    BackendGroups: []BackendGroup{{ID: "api-servers"}},
    Telemetry: TelemetryConfig{MetricsEnabled: true},
}
```

---

```go
// ServerConfig defines proxy server settings
type ServerConfig struct {
    Address  string `yaml:"address"`   // Listen address (e.g., "0.0.0.0:443")
    CertFile string `yaml:"cert_file"` // Path to TLS certificate
    KeyFile  string `yaml:"key_file"`  // Path to TLS private key
    
    // Optional: QUIC-specific settings
    MaxIdleTimeout    time.Duration `yaml:"max_idle_timeout"`
    MaxIncomingStreams int64         `yaml:"max_incoming_streams"`
    KeepAlivePeriod   time.Duration `yaml:"keep_alive_period"`
}
```

**Explanation**:
- **Address**: Where the proxy listens (IP:Port)
  - `0.0.0.0:443` = listen on all interfaces, port 443
  - `127.0.0.1:8443` = only local connections, port 8443
- **CertFile/KeyFile**: TLS is mandatory for QUIC
- **MaxIdleTimeout**: How long to keep idle connections alive
  - Too short: Frequent reconnections (overhead)
  - Too long: Resource waste on idle connections
- **MaxIncomingStreams**: Max concurrent requests per connection
  - QUIC multiplexes multiple requests on one connection
  - This limits how many to prevent resource exhaustion
- **KeepAlivePeriod**: Send keep-alive pings to detect dead connections
  - If no response, close the connection

---

```go
// RouteConfig defines URL routing rules
type RouteConfig struct {
    ID           string   `yaml:"id"`            // Unique route identifier
    Path         string   `yaml:"path"`          // URL path pattern (e.g., "/api/*")
    Methods      []string `yaml:"methods"`       // HTTP methods (GET, POST, etc.)
    BackendGroup string   `yaml:"backend_group"` // Which backend group to use
    
    // Optional: Advanced routing
    Headers      map[string]string `yaml:"headers"`       // Match request headers
    StripPrefix  string            `yaml:"strip_prefix"`  // Remove prefix before forwarding
    AddHeaders   map[string]string `yaml:"add_headers"`   // Add headers to backend request
}
```

**Explanation**:
- **Path patterns**:
  - `/api/*` matches `/api/users`, `/api/orders`, etc.
  - `/static/*` matches `/static/css/style.css`
  - Exact match: `/health` only matches `/health`
- **Methods**: Restrict routes to specific HTTP methods
  - Example: `methods: [GET, POST]` rejects DELETE requests
- **StripPrefix**: Remove part of the URL before forwarding
  - Client requests: `/api/v1/users`
  - `strip_prefix: "/api/v1"`
  - Backend receives: `/users`
  - Useful when backend has different URL structure
- **AddHeaders**: Inject headers into backend request
  - Example: `X-Forwarded-For: client-ip`
  - Backend knows original client IP

**Example route**:
```yaml
routes:
  - id: "api-route"
    path: "/api/*"
    methods: ["GET", "POST", "PUT", "DELETE"]
    backend_group: "api-servers"
    strip_prefix: "/api"
    add_headers:
      X-Proxy-Version: "1.0"
      X-Request-ID: "${request_id}"
```

---

```go
// BackendGroup defines a group of backend servers
type BackendGroup struct {
    ID          string           `yaml:"id"`
    Strategy    string           `yaml:"strategy"`  // Load balancing strategy
    Backends    []BackendConfig  `yaml:"backends"`
    HealthCheck *HealthCheckConfig `yaml:"health_check"`
    
    // Optional: Advanced settings
    ConnectionPool ConnectionPoolConfig `yaml:"connection_pool"`
    Timeout        TimeoutConfig        `yaml:"timeout"`
    Retry          RetryConfig          `yaml:"retry"`
}
```

**Explanation**:
- **Strategy**: How to distribute load
  - `round_robin`: Fair distribution (A → B → C → A...)
  - `least_connections`: Send to server with fewest active connections
  - `random`: Random selection (simple, effective)
  - `weighted`: Consider backend weights
- **HealthCheck**: Monitor backend availability
  - If disabled, proxy assumes all backends are healthy
  - Can lead to errors if a backend dies
- **ConnectionPool**: Reuse connections for performance
  - Max idle connections to keep open
  - Timeout for idle connections
- **Timeout**: How long to wait for backend response
  - Connect timeout: Time to establish connection
  - Request timeout: Time to receive full response
- **Retry**: Automatic retry on failure
  - Number of attempts
  - Which status codes to retry
  - Backoff strategy (exponential, linear)

---

```go
// BackendConfig defines a single backend server
type BackendConfig struct {
    URL     string `yaml:"url"`     // Backend URL (http://10.0.0.1:8080)
    Weight  int    `yaml:"weight"`  // Weight for weighted load balancing
    Healthy bool   `yaml:"-"`       // Runtime health status (not in YAML)
}
```

**Explanation**:
- **URL**: Full backend address
  - `http://10.0.0.1:8080` = internal HTTP server
  - `https://backend.internal:9000` = internal HTTPS server
- **Weight**: For weighted load balancing
  - Weight 100 = normal
  - Weight 200 = receives 2x traffic
  - Weight 50 = receives 0.5x traffic
  - Use case: Newer/more powerful servers get more traffic
- **Healthy**: Runtime flag (not configurable)
  - `yaml:"-"` means "don't save this to YAML"
  - Set by health checker at runtime
  - Used by load balancer to skip unhealthy backends

---

```go
// HealthCheckConfig defines health check parameters
type HealthCheckConfig struct {
    Enabled            bool          `yaml:"enabled"`
    Interval           time.Duration `yaml:"interval"`            // Check frequency
    Timeout            time.Duration `yaml:"timeout"`             // Max wait time
    Path               string        `yaml:"path"`                // Endpoint to check
    HealthyThreshold   int           `yaml:"healthy_threshold"`   // Successes needed
    UnhealthyThreshold int           `yaml:"unhealthy_threshold"` // Failures needed
    
    // Active vs Passive
    Passive PassiveHealthCheck `yaml:"passive"`
}

type PassiveHealthCheck struct {
    Enabled           bool          `yaml:"enabled"`
    MaxFailures       int           `yaml:"max_failures"`
    ObservationWindow time.Duration `yaml:"observation_window"`
}
```

**Explanation**:

**Active Health Checks**:
- Proxy **actively sends** HTTP requests to backends
- Example: Every 10 seconds, send `GET /health`
- If backend responds with 200 OK → Healthy
- If no response or error → Unhealthy

**Why thresholds?**
- **HealthyThreshold**: Need 2 consecutive successes to mark healthy
  - Prevents flapping (healthy → unhealthy → healthy rapidly)
  - One success might be a fluke
- **UnhealthyThreshold**: Need 3 consecutive failures to mark unhealthy
  - Tolerates temporary network blips
  - Reduces false negatives

**Example scenario**:
```
Time 0s:  Check → Success (success_count=1)
Time 10s: Check → Success (success_count=2) → MARK HEALTHY ✅
Time 20s: Check → Fail (fail_count=1)
Time 30s: Check → Fail (fail_count=2)
Time 40s: Check → Fail (fail_count=3) → MARK UNHEALTHY ❌
Time 50s: Check → Success (success_count=1)
Time 60s: Check → Success (success_count=2) → MARK HEALTHY ✅
```

**Passive Health Checks**:
- Monitor **real traffic** for errors
- Don't send separate health check requests
- Track error rate over time window

**Example**:
```yaml
passive:
  enabled: true
  max_failures: 5              # 5 consecutive errors = unhealthy
  observation_window: 60s      # Reset counter after 60s of success
```

**Scenario**:
```
Request 1 → Backend → 200 OK ✅ (error_count=0)
Request 2 → Backend → 500 Error ❌ (error_count=1)
Request 3 → Backend → 502 Bad Gateway ❌ (error_count=2)
Request 4 → Backend → 200 OK ✅ (error_count=0, reset!)
Request 5 → Backend → 500 Error ❌ (error_count=1)
Request 6 → Backend → 500 Error ❌ (error_count=2)
Request 7 → Backend → 500 Error ❌ (error_count=3)
Request 8 → Backend → 500 Error ❌ (error_count=4)
Request 9 → Backend → 500 Error ❌ (error_count=5) → MARK UNHEALTHY ❌
```

**Active vs Passive comparison**:
| Aspect | Active | Passive |
|--------|--------|---------|
| **Traffic overhead** | Adds health check requests | No extra requests |
| **Detection speed** | Fast (dedicated checks) | Slower (needs real traffic) |
| **Accuracy** | Can miss load-related issues | Reflects real user experience |
| **Use case** | Critical systems | High-traffic systems |

**Best practice**: Use both together!

---

```go
// LoadConfig reads and parses the configuration file
func LoadConfig(filename string) (*Config, error) {
    // Step 1: Read file contents
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file: %w", err)
    }
```

**Explanation**:
- `os.ReadFile()` reads entire file into memory
- Returns byte slice: `[]byte`
- Error if file doesn't exist or can't be read
- `%w` wraps error for better error messages

```go
    // Step 2: Parse YAML into Config struct
    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }
```

**Explanation**:
- `yaml.Unmarshal()` converts YAML bytes to Go struct
- Uses struct tags (`yaml:"server"`) to map fields
- Returns error if YAML is malformed or types don't match

**Example error**:
```yaml
server:
  max_idle_timeout: "not a duration"  # Should be "30s" or "1m"
```
Error: `cannot unmarshal !!str 'not a duration' into time.Duration`

```go
    // Step 3: Set defaults for optional fields
    cfg.applyDefaults()
```

**Explanation**:
- If user doesn't specify some values, use sensible defaults
- Example: No timeout specified → Use 30s default

```go
    // Step 4: Validate configuration
    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }
    
    return &cfg, nil
}
```

**Explanation**:
- Checks logical consistency
- Ensures required fields are present
- Validates value ranges

---

```go
// applyDefaults sets default values for optional fields
func (c *Config) applyDefaults() {
    // Server defaults
    if c.Server.MaxIdleTimeout == 0 {
        c.Server.MaxIdleTimeout = 30 * time.Second
    }
    if c.Server.MaxIncomingStreams == 0 {
        c.Server.MaxIncomingStreams = 100
    }
    if c.Server.KeepAlivePeriod == 0 {
        c.Server.KeepAlivePeriod = 15 * time.Second
    }
    
    // Health check defaults
    for i := range c.BackendGroups {
        if c.BackendGroups[i].HealthCheck != nil {
            hc := c.BackendGroups[i].HealthCheck
            if hc.Interval == 0 {
                hc.Interval = 10 * time.Second
            }
            if hc.Timeout == 0 {
                hc.Timeout = 5 * time.Second
            }
            if hc.Path == "" {
                hc.Path = "/health"
            }
            if hc.HealthyThreshold == 0 {
                hc.HealthyThreshold = 2
            }
            if hc.UnhealthyThreshold == 0 {
                hc.UnhealthyThreshold = 3
            }
        }
    }
}
```

**Explanation**:
- **Why defaults?** User convenience - don't force specifying every option
- **Why these values?** Based on industry best practices
  - 30s idle timeout: Balance between resource usage and reconnection overhead
  - 100 streams: Reasonable for most applications
  - 10s health check interval: Catches failures within 30s
  - 5s health check timeout: Long enough for slow responses

---

```go
// Validate checks configuration for errors
func (c *Config) Validate() error {
    // Must have at least one route
    if len(c.Routes) == 0 {
        return fmt.Errorf("at least one route must be configured")
    }
    
    // Must have at least one backend group
    if len(c.BackendGroups) == 0 {
        return fmt.Errorf("at least one backend group must be configured")
    }
    
    // Check each route references a valid backend group
    groupIDs := make(map[string]bool)
    for _, group := range c.BackendGroups {
        groupIDs[group.ID] = true
        
        // Each group must have at least one backend
        if len(group.Backends) == 0 {
            return fmt.Errorf("backend group '%s' has no backends", group.ID)
        }
        
        // Validate load balancing strategy
        validStrategies := []string{"round_robin", "least_connections", "random", "weighted"}
        valid := false
        for _, s := range validStrategies {
            if group.Strategy == s {
                valid = true
                break
            }
        }
        if !valid {
            return fmt.Errorf("invalid strategy '%s' for group '%s'", group.Strategy, group.ID)
        }
    }
    
    // Validate route references
    for _, route := range c.Routes {
        if !groupIDs[route.BackendGroup] {
            return fmt.Errorf("route '%s' references non-existent backend group '%s'", 
                route.ID, route.BackendGroup)
        }
    }
    
    // Validate TLS certificate files exist
    if _, err := os.Stat(c.Server.CertFile); os.IsNotExist(err) {
        return fmt.Errorf("certificate file not found: %s", c.Server.CertFile)
    }
    if _, err := os.Stat(c.Server.KeyFile); os.IsNotExist(err) {
        return fmt.Errorf("key file not found: %s", c.Server.KeyFile)
    }
    
    return nil
}
```

**Explanation**:
- **Why validate?** Catch errors early (before starting server)
- **What we check**:
  - Required fields present
  - References valid (routes point to existing backend groups)
  - Files exist (certificates)
  - Values in valid ranges
  - Enum fields have valid values

**Example validation errors**:
```
Error: route 'api-route' references non-existent backend group 'api-servers'
→ User typo: 'api-servers' should be 'api-server' (no 's')

Error: backend group 'static' has no backends
→ User forgot to add backend URLs

Error: invalid strategy 'round-robin' for group 'api'
→ Should be 'round_robin' (underscore, not hyphen)

Error: certificate file not found: certs/server.crt
→ Need to run 'make certs' first
```

---

### `internal/proxy/` - Core Proxy Logic

This is where the magic happens - the actual HTTP/3 proxy implementation.

#### `internal/proxy/server.go` - Main Server

**What it does**:
- Listens for QUIC/HTTP/3 connections
- Handles TLS handshake
- Routes requests to appropriate backends
- Manages connection lifecycle

**Code walkthrough**:

```go
package proxy

import (
    "context"
    "crypto/tls"
    "fmt"
    "net/http"
    
    "github.com/quic-go/quic-go"
    "github.com/quic-go/quic-go/http3"
)

// Server represents the QUIC reverse proxy server
type Server struct {
    config         *config.Config
    backendManager *backend.Manager
    router         *Router
    quicServer     *http3.Server
    tlsConfig      *tls.Config
}
```

**Explanation**:
- **config**: User configuration (loaded from YAML)
- **backendManager**: Manages backend connections and pools
- **router**: Matches incoming requests to routes
- **quicServer**: The actual HTTP/3 server from quic-go library
- **tlsConfig**: TLS settings (certificates, cipher suites, etc.)

---

```go
// NewServer creates a new proxy server instance
func NewServer(cfg *config.Config, bm *backend.Manager) (*Server, error) {
    // Step 1: Load TLS certificate and key
    cert, err := tls.LoadX509KeyPair(cfg.Server.CertFile, cfg.Server.KeyFile)
    if err != nil {
        return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
    }
```

**Explanation**:
- `tls.LoadX509KeyPair()` reads certificate and private key files
- Returns a `tls.Certificate` object
- Error if files don't exist, are corrupted, or don't match

**What is X509?**
- X.509 is the standard format for TLS certificates
- Contains: Public key, domain name, issuer, validity dates
- Paired with private key (must be kept secret)

---

```go
    // Step 2: Configure TLS settings
    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        MinVersion:   tls.VersionTLS13,  // QUIC requires TLS 1.3+
        NextProtos:   []string{"h3"},    // Advertise HTTP/3 support
    }
```

**Explanation**:
- **Certificates**: The cert we just loaded
- **MinVersion**: TLS 1.3 is required for QUIC
  - TLS 1.3 is faster (1-RTT handshake)
  - More secure (removes weak ciphers)
  - QUIC spec mandates it
- **NextProtos**: Application-Layer Protocol Negotiation (ALPN)
  - Client says: "I support h3 (HTTP/3)"
  - Server says: "I also support h3"
  - They agree to use HTTP/3

**Why is this needed?**
- Same TLS connection can run different protocols
- ALPN negotiates which protocol to use
- Example: `h3` (HTTP/3), `h2` (HTTP/2), `http/1.1`

---

```go
    // Step 3: Create router for request matching
    router := NewRouter(cfg.Routes)
```

**Explanation**:
- Router maps URL paths to backend groups
- Example: `/api/*` → backend group "api-servers"
- Uses pattern matching (not exact string matching)

---

```go
    // Step 4: Create HTTP/3 server
    server := &Server{
        config:         cfg,
        backendManager: bm,
        router:         router,
        tlsConfig:      tlsConfig,
    }
    
    // Create the underlying QUIC/HTTP/3 server
    server.quicServer = &http3.Server{
        Addr:      cfg.Server.Address,
        Handler:   server,  // Server implements http.Handler interface
        TLSConfig: tlsConfig,
        QUICConfig: &quic.Config{
            MaxIdleTimeout:                 cfg.Server.MaxIdleTimeout,
            MaxIncomingStreams:             cfg.Server.MaxIncomingStreams,
            KeepAlivePeriod:                cfg.Server.KeepAlivePeriod,
            EnableDatagrams:                false,  // We don't use QUIC datagrams
            Allow0RTT:                      true,   // Enable 0-RTT resumption
        },
    }
    
    return server, nil
}
```

**Explanation**:
- **Handler: server**: When a request arrives, call `server.ServeHTTP()`
  - This is where we implement the proxy logic
- **QUICConfig**: QUIC-specific settings
  - **MaxIdleTimeout**: Close idle connections after this time
  - **MaxIncomingStreams**: Max concurrent requests per connection
  - **KeepAlivePeriod**: Send PING frames to detect dead connections
  - **Allow0RTT**: Enable zero-round-trip resumption
    - First connection: 1-RTT (one round trip)
    - Subsequent connections: 0-RTT (immediate data)
    - Security note: 0-RTT data can be replayed, so only safe for idempotent requests

---

```go
// ServeHTTP implements the http.Handler interface
// This method is called for every incoming HTTP request
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Step 1: Log incoming request
    logger.Info("Incoming request",
        "method", r.Method,
        "path", r.URL.Path,
        "remote", r.RemoteAddr,
    )
```

**Explanation**:
- **w**: Response writer (where we write the response)
- **r**: Incoming request (method, URL, headers, body)
- Log every request for debugging and monitoring

---

```go
    // Step 2: Find matching route
    route := s.router.Match(r)
    if route == nil {
        // No route matches this request
        http.Error(w, "No route found", http.StatusNotFound)
        return
    }
```

**Explanation**:
- Router checks all configured routes
- Returns the first matching route
- If no match, return 404 Not Found

**Example**:
```
Request: GET /api/users
Routes:
  1. /static/* → No match
  2. /api/* → Match! ✅
  3. /admin/* → (Not checked, already matched)
  
Route selected: /api/* → backend group "api-servers"
```

---

```go
    // Step 3: Get backend group for this route
    backendGroup := s.backendManager.GetGroup(route.BackendGroup)
    if backendGroup == nil {
        http.Error(w, "Backend group not found", http.StatusInternalServerError)
        return
    }
```

**Explanation**:
- Backend manager maintains all backend groups
- Get the group specified in the matched route
- If group doesn't exist (config error), return 500

---

```go
    // Step 4: Select a healthy backend using load balancer
    backend := backendGroup.NextBackend()
    if backend == nil {
        // All backends are unhealthy!
        http.Error(w, "No healthy backends available", http.StatusServiceUnavailable)
        return
    }
```

**Explanation**:
- Load balancer selects one backend from the group
- Selection algorithm depends on strategy (round robin, least connections, etc.)
- **Only selects healthy backends** (health checker marks them)
- If all backends unhealthy, return 503 Service Unavailable

**Example scenario**:
```
Backend Group "api-servers":
  - Backend 1: Healthy ✅
  - Backend 2: Unhealthy ❌ (health check failed)
  - Backend 3: Healthy ✅

Load Balancer (Round Robin):
  Request 1 → Backend 1 (skip Backend 2, it's unhealthy)
  Request 2 → Backend 3
  Request 3 → Backend 1
  Request 4 → Backend 3
```

---

```go
    // Step 5: Create backend request
    backendURL := backend.URL + r.URL.Path
    if r.URL.RawQuery != "" {
        backendURL += "?" + r.URL.RawQuery
    }
    
    backendReq, err := http.NewRequestWithContext(r.Context(), r.Method, backendURL, r.Body)
    if err != nil {
        http.Error(w, "Failed to create backend request", http.StatusInternalServerError)
        return
    }
```

**Explanation**:
- **Construct backend URL**:
  - Backend: `http://10.0.0.1:8080`
  - Request path: `/api/users`
  - Query: `?limit=10`
  - Result: `http://10.0.0.1:8080/api/users?limit=10`
- **NewRequestWithContext**: Create HTTP request with cancellation support
  - If client disconnects, cancel backend request too
  - Prevents wasted backend processing

---

```go
    // Step 6: Copy headers from client request to backend request
    // (Preserve client's headers)
    for key, values := range r.Header {
        for _, value := range values {
            backendReq.Header.Add(key, value)
        }
    }
    
    // Add proxy-specific headers
    backendReq.Header.Set("X-Forwarded-For", r.RemoteAddr)
    backendReq.Header.Set("X-Forwarded-Proto", "https")
    backendReq.Header.Set("X-Real-IP", r.RemoteAddr)
```

**Explanation**:
- **Copy all headers**: Client's headers are preserved
  - Example: `Authorization: Bearer token123`
  - Backend receives the same token
- **Add proxy headers**: Inform backend about the original request
  - **X-Forwarded-For**: Original client IP
  - **X-Forwarded-Proto**: Original protocol (QUIC/HTTPS)
  - **X-Real-IP**: Another way to pass client IP

**Why do backends need this?**
- Backend sees proxy IP, not client IP
- Backends need client IP for: rate limiting, geolocation, logging
- X-Forwarded-For preserves the original client IP

**Example**:
```
Client: 203.0.113.45 → Proxy: 10.0.0.10 → Backend: 10.0.0.1

Without X-Forwarded-For:
Backend sees: RemoteAddr = 10.0.0.10 (proxy IP)

With X-Forwarded-For:
Backend sees: X-Forwarded-For = 203.0.113.45 (client IP)
```

---

```go
    // Step 7: Send request to backend
    startTime := time.Now()
    
    resp, err := backend.Client.Do(backendReq)
    if err != nil {
        // Backend request failed
        logger.Error("Backend request failed",
            "backend", backend.URL,
            "error", err,
        )
        
        // Mark backend as potentially unhealthy (passive health check)
        backendGroup.RecordFailure(backend)
        
        http.Error(w, "Backend request failed", http.StatusBadGateway)
        return
    }
    defer resp.Body.Close()
    
    duration := time.Since(startTime)
```

**Explanation**:
- **backend.Client.Do()**: Send HTTP request to backend
  - Uses connection pool (reuses connections)
  - Times out after configured duration
- **Error handling**: If backend fails
  - Log the error for debugging
  - Record failure (passive health check)
  - Return 502 Bad Gateway to client
- **Track duration**: Measure backend response time
  - Used for latency metrics
  - Helps identify slow backends

---

```go
    // Step 8: Copy response headers from backend to client
    for key, values := range resp.Header {
        for _, value := range values {
            w.Header().Add(key, value)
        }
    }
    
    // Add proxy identification header
    w.Header().Set("Via", "QUIC-Reverse-Proxy/1.0")
```

**Explanation**:
- Copy all backend response headers to client
- **Via**: Indicates the response went through a proxy
  - Standard HTTP header (RFC 7230)
  - Helps debug multi-proxy chains

---

```go
    // Step 9: Write status code
    w.WriteHeader(resp.StatusCode)
    
    // Step 10: Stream response body from backend to client
    bytesWritten, err := io.Copy(w, resp.Body)
    if err != nil {
        logger.Error("Failed to write response", "error", err)
        return
    }
```

**Explanation**:
- **WriteHeader**: Set HTTP status code (200, 404, 500, etc.)
- **io.Copy**: Stream response body
  - Reads from `resp.Body` (backend)
  - Writes to `w` (client)
  - Does NOT load entire response into memory
  - Efficient for large responses (videos, files)

**Why streaming?**
```
Without streaming (bad):
Proxy reads 1GB video → Stores in memory → Sends to client
Memory usage: 1GB

With streaming (good):
Proxy reads 64KB → Sends to client → Reads next 64KB → Sends...
Memory usage: 64KB
```

---

```go
    // Step 11: Record metrics
    s.recordMetrics(MetricsData{
        Method:       r.Method,
        Path:         r.URL.Path,
        Backend:      backend.URL,
        StatusCode:   resp.StatusCode,
        Duration:     duration,
        BytesWritten: bytesWritten,
    })
    
    logger.Info("Request completed successfully",
        "method", r.Method,
        "path", r.URL.Path,
        "backend", backend.URL,
        "status", resp.StatusCode,
        "duration_ms", duration.Milliseconds(),
        "bytes", bytesWritten,
    )
}
```

**Explanation**:
- **Record metrics**: Send data to Prometheus
  - Request count by method, path, status
  - Latency histogram
  - Bytes transferred
- **Log completion**: Structured log entry
  - Can be parsed by log aggregators (ELK, Splunk)
  - Includes all important metadata

**Prometheus metrics created**:
```prometheus
# Request counter
http_requests_total{method="GET",path="/api/users",backend="backend1",status="200"} 1523

# Latency histogram
http_request_duration_seconds_bucket{le="0.1"} 1200
http_request_duration_seconds_bucket{le="0.5"} 1500

# Bytes transferred
http_response_bytes_total{backend="backend1"} 15728640
```

---

### Summary of Request Flow

Let's trace a complete request through the system:

```
1. Client sends: GET https://proxy.example.com/api/users?limit=10
   └→ QUIC connection established (or resumed with 0-RTT)

2. ServeHTTP() called with request

3. Router matches:
   └→ Route: /api/* → Backend Group: "api-servers"

4. Backend Manager gets group:
   └→ Group: "api-servers" (3 backends)

5. Load Balancer selects backend:
   └→ Strategy: Least Connections
   └→ Backend 1: 10 connections
   └→ Backend 2: 5 connections ✅ (selected)
   └→ Backend 3: 8 connections

6. Health check verifies:
   └→ Backend 2: Healthy ✅

7. Create backend request:
   └→ URL: http://10.0.0.2:8002/api/users?limit=10
   └→ Copy headers from client
   └→ Add: X-Forwarded-For, X-Real-IP

8. Send request to backend:
   └→ Start timer
   └→ HTTP Client (with connection pool)
   └→ Backend processes request

9. Receive backend response:
   └→ Status: 200 OK
   └→ Duration: 45ms
   └→ Body: JSON data (25KB)

10. Stream response to client:
    └→ Copy headers
    └→ Add: Via: QUIC-Reverse-Proxy/1.0
    └→ Write status code: 200
    └→ Stream body (25KB)

11. Record metrics:
    └→ Prometheus: request_count++, latency=45ms
    └→ OpenTelemetry: Trace with spans
    └→ Log: INFO request completed

12. Client receives response
    └→ Total time: 50ms (including network)
    └→ QUIC connection kept alive for future requests
```

---

#### `internal/proxy/router.go` - Request Router

**What it does**: Matches incoming requests to configured routes based on URL patterns.

**Code walkthrough**:

```go
package proxy

import (
    "net/http"
    "strings"
    "path"
)

// Router matches HTTP requests to routes
type Router struct {
    routes []*Route
}

// Route represents a configured routing rule
type Route struct {
    ID           string
    PathPattern  string
    Methods      []string
    BackendGroup string
    StripPrefix  string
    AddHeaders   map[string]string
}
```

**Explanation**:
- **routes**: List of all configured routes (order matters!)
- **PathPattern**: URL pattern to match (supports wildcards)
- **Methods**: Allowed HTTP methods (if empty, allow all)

---

```go
// NewRouter creates a router from configuration
func NewRouter(configs []config.RouteConfig) *Router {
    routes := make([]*Route, 0, len(configs))
    
    for _, cfg := range configs {
        route := &Route{
            ID:           cfg.ID,
            PathPattern:  cfg.Path,
            Methods:      cfg.Methods,
            BackendGroup: cfg.BackendGroup,
            StripPrefix:  cfg.StripPrefix,
            AddHeaders:   cfg.AddHeaders,
        }
        routes = append(routes, route)
    }
    
    return &Router{routes: routes}
}
```

**Explanation**:
- Converts configuration into runtime Route objects
- Preserves route order (first match wins)

---

```go
// Match finds the first route matching the request
func (r *Router) Match(req *http.Request) *Route {
    for _, route := range r.routes {
        // Check if path matches
        if !r.matchPath(route.PathPattern, req.URL.Path) {
            continue
        }
        
        // Check if method matches (if methods specified)
        if len(route.Methods) > 0 {
            methodAllowed := false
            for _, method := range route.Methods {
                if method == req.Method {
                    methodAllowed = true
                    break
                }
            }
            if !methodAllowed {
                continue
            }
        }
        
        // Match found!
        return route
    }
    
    // No route matched
    return nil
}
```

**Explanation**:
- Iterates through routes in order
- First checks path pattern match
- Then checks if HTTP method is allowed
- Returns first matching route (or nil)

**Example matching**:
```
Routes:
  1. Path: /api/*, Methods: [GET, POST]
  2. Path: /admin/*, Methods: [GET]
  3. Path: /*, Methods: []

Request: POST /api/users
  Route 1: Path matches /api/* ✅, Method POST allowed ✅ → MATCH!
  (Routes 2 and 3 not checked)

Request: DELETE /api/users
  Route 1: Path matches /api/* ✅, Method DELETE not allowed ❌
  Route 2: Path doesn't match /admin/* ❌
  Route 3: Path matches /* ✅, No method restriction ✅ → MATCH!

Request: GET /unknown
  Route 1: Path doesn't match /api/* ❌
  Route 2: Path doesn't match /admin/* ❌
  Route 3: Path matches /* ✅ → MATCH!
```

---

```go
// matchPath checks if a path matches a pattern
func (r *Router) matchPath(pattern, requestPath string) bool {
    // Exact match (no wildcard)
    if !strings.Contains(pattern, "*") {
        return pattern == requestPath
    }
    
    // Wildcard match
    // Pattern: /api/* should match /api/users, /api/orders, etc.
    prefix := strings.TrimSuffix(pattern, "/*")
    
    // Check if request path starts with prefix
    if !strings.HasPrefix(requestPath, prefix) {
        return false
    }
    
    // For pattern /api/*, path /api should NOT match (needs something after)
    // But /api/ and /api/users should match
    if requestPath == prefix {
        return false
    }
    
    if len(requestPath) > len(prefix) && requestPath[len(prefix)] != '/' {
        return false
    }
    
    return true
}
```

**Explanation**:
- **Exact match**: `/health` only matches `/health`
- **Wildcard match**: `/api/*` matches `/api/users`, `/api/v1/orders`, etc.

**Matching examples**:
```
Pattern: /api/*
  /api/users → Match ✅
  /api/v1/users → Match ✅
  /api → No match ❌ (needs something after /api)
  /api/ → Match ✅
  /apiv2/users → No match ❌ (doesn't start with /api/)

Pattern: /health
  /health → Match ✅
  /health/ → No match ❌
  /healthcheck → No match ❌
```

---

### `internal/backend/` - Backend Management

This package manages backend servers, connection pools, and load balancing.

#### `internal/backend/manager.go` - Backend Manager

**What it does**: 
- Manages all backend groups
- Maintains connection pools for each backend
- Coordinates health checking

**Code walkthrough**:

```go
package backend

import (
    "net/http"
    "sync"
    "time"
)

// Manager manages all backend groups and their connection pools
type Manager struct {
    groups map[string]*BackendGroup  // Map: group ID → backend group
    mu     sync.RWMutex               // Protects concurrent access
}

// BackendGroup represents a group of backend servers
type BackendGroup struct {
    ID        string
    Backends  []*Backend
    Balancer  LoadBalancer
    HealthCheck *HealthChecker
    mu        sync.RWMutex
}

// Backend represents a single backend server
type Backend struct {
    URL              string
    Weight           int
    Healthy          bool
    ActiveConnections int32  // Atomic counter
    Client           *http.Client  // HTTP client with connection pool
    
    // Health check stats
    ConsecutiveFailures int
    ConsecutiveSuccesses int
    LastHealthCheck     time.Time
}
```

**Explanation**:
- **Manager**: Central registry of all backend groups
- **BackendGroup**: A logical group of backends (e.g., "api-servers", "static-servers")
- **Backend**: Individual server with its own connection pool
- **mu sync.RWMutex**: Allows multiple concurrent readers, but exclusive writers
  - Multiple requests can read backend list simultaneously
  - Health checker can update status exclusively

**Why RWMutex?**
```go
// Multiple goroutines can read simultaneously
func (m *Manager) GetGroup(id string) *BackendGroup {
    m.mu.RLock()  // Read lock (shared)
    defer m.mu.RUnlock()
    return m.groups[id]
}

// Only one goroutine can write at a time
func (m *Manager) UpdateHealth(backend *Backend, healthy bool) {
    m.mu.Lock()  // Write lock (exclusive)
    defer m.mu.Unlock()
    backend.Healthy = healthy
}
```

---

```go
// NewManager creates a new backend manager
func NewManager(configs []config.BackendGroup) *Manager {
    manager := &Manager{
        groups: make(map[string]*BackendGroup),
    }
    
    for _, cfg := range configs {
        group := manager.createGroup(cfg)
        manager.groups[cfg.ID] = group
    }
    
    return manager
}
```

**Explanation**:
- Creates manager with empty groups map
- Converts each config into a runtime BackendGroup
- Stores groups by ID for quick lookup

---

```go
// createGroup initializes a backend group from configuration
func (m *Manager) createGroup(cfg config.BackendGroup) *BackendGroup {
    // Create backends with connection pools
    backends := make([]*Backend, 0, len(cfg.Backends))
    
    for _, backendCfg := range cfg.Backends {
        // Create HTTP client with connection pooling
        client := &http.Client{
            Transport: &http.Transport{
                MaxIdleConns:        100,  // Total idle connections
                MaxIdleConnsPerHost: 10,   // Idle connections per backend
                IdleConnTimeout:     90 * time.Second,
                DisableKeepAlives:   false,  // Enable keep-alive
                DisableCompression:  false,  // Allow compression
            },
            Timeout: 30 * time.Second,  // Total request timeout
        }
        
        backend := &Backend{
            URL:     backendCfg.URL,
            Weight:  backendCfg.Weight,
            Healthy: true,  // Assume healthy until proven otherwise
            Client:  client,
        }
        
        backends = append(backends, backend)
    }
    
    // Create load balancer based on strategy
    var balancer LoadBalancer
    switch cfg.Strategy {
    case "round_robin":
        balancer = NewRoundRobinBalancer(backends)
    case "least_connections":
        balancer = NewLeastConnectionsBalancer(backends)
    case "random":
        balancer = NewRandomBalancer(backends)
    case "weighted":
        balancer = NewWeightedBalancer(backends)
    default:
        balancer = NewRoundRobinBalancer(backends)
    }
    
    return &BackendGroup{
        ID:       cfg.ID,
        Backends: backends,
        Balancer: balancer,
    }
}
```

**Explanation**:

**Connection Pool Configuration**:
- **MaxIdleConns: 100**: Keep up to 100 idle connections across all backends
  - Idle = not currently in use, but kept open for reuse
  - Why? Reusing connections is faster than creating new ones
- **MaxIdleConnsPerHost: 10**: Max 10 idle connections to each backend
  - Prevents one backend from hogging all idle connections
- **IdleConnTimeout: 90s**: Close idle connections after 90 seconds
  - Balance: Keep connections alive, but not forever (resource waste)
- **DisableKeepAlives: false**: Allow HTTP keep-alive
  - Keep-alive = reuse TCP connection for multiple requests
  - Without: New TCP connection for every request (slow!)

**Connection Pool Benefits**:
```
Without connection pooling:
Request 1: Open TCP → TLS handshake → HTTP request → Close (150ms)
Request 2: Open TCP → TLS handshake → HTTP request → Close (150ms)
Request 3: Open TCP → TLS handshake → HTTP request → Close (150ms)
Total: 450ms

With connection pooling:
Request 1: Open TCP → TLS handshake → HTTP request → Keep alive (150ms)
Request 2: Reuse connection → HTTP request → Keep alive (50ms)
Request 3: Reuse connection → HTTP request → Keep alive (50ms)
Total: 250ms (44% faster!)
```

---

```go
// GetGroup returns a backend group by ID
func (m *Manager) GetGroup(id string) *BackendGroup {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.groups[id]
}
```

**Explanation**:
- Thread-safe read access
- RLock allows concurrent reads
- Fast O(1) lookup by ID

---

```go
// NextBackend selects a healthy backend using load balancing
func (g *BackendGroup) NextBackend() *Backend {
    g.mu.RLock()
    defer g.mu.RUnlock()
    
    // Filter healthy backends
    healthyBackends := make([]*Backend, 0, len(g.Backends))
    for _, backend := range g.Backends {
        if backend.Healthy {
            healthyBackends = append(healthyBackends, backend)
        }
    }
    
    // No healthy backends available
    if len(healthyBackends) == 0 {
        return nil
    }
    
    // Use load balancer to select from healthy backends
    return g.Balancer.Next(healthyBackends)
}
```

**Explanation**:
- **Filter step**: Only consider healthy backends
  - Health checker marks backends as healthy/unhealthy
- **Load balancer**: Selects one backend from healthy set
  - Algorithm depends on strategy (round robin, least connections, etc.)

**Example scenario**:
```
Backend Group: "api-servers"
  Backend 1: Healthy ✅
  Backend 2: Unhealthy ❌ (health check failed)
  Backend 3: Healthy ✅
  Backend 4: Healthy ✅

Filter step:
  healthyBackends = [Backend 1, Backend 3, Backend 4]

Load Balancer (Round Robin):
  Request 1 → Backend 1
  Request 2 → Backend 3
  Request 3 → Backend 4
  Request 4 → Backend 1 (cycle repeats)
```

---

#### `internal/backend/balancer.go` - Load Balancers

**What it does**: Implements different load balancing algorithms.

**Code walkthrough**:

```go
package backend

import (
    "math/rand"
    "sync/atomic"
)

// LoadBalancer interface that all strategies must implement
type LoadBalancer interface {
    Next(backends []*Backend) *Backend
}
```

**Explanation**:
- Interface allows swapping strategies easily
- Each strategy implements `Next()` method
- Takes list of healthy backends, returns one selected backend

---

##### Round Robin Balancer

```go
// RoundRobinBalancer distributes requests evenly in order
type RoundRobinBalancer struct {
    current uint32  // Atomic counter
}

func NewRoundRobinBalancer(backends []*Backend) *RoundRobinBalancer {
    return &RoundRobinBalancer{}
}

func (rr *RoundRobinBalancer) Next(backends []*Backend) *Backend {
    if len(backends) == 0 {
        return nil
    }
    
    // Atomically increment counter and get backend
    idx := atomic.AddUint32(&rr.current, 1) % uint32(len(backends))
    return backends[idx]
}
```

**Explanation**:
- **atomic.AddUint32**: Thread-safe increment
  - Multiple goroutines can call Next() simultaneously
  - Atomic ensures each gets a unique number
- **Modulo (%)**: Wraps around to 0 after reaching end

**How it works**:
```
Backends: [A, B, C]

Request 1: current=0 → 0 % 3 = 0 → Backend A
Request 2: current=1 → 1 % 3 = 1 → Backend B
Request 3: current=2 → 2 % 3 = 2 → Backend C
Request 4: current=3 → 3 % 3 = 0 → Backend A (cycle)
Request 5: current=4 → 4 % 3 = 1 → Backend B
```

**Pros**:
- ✅ Simple and predictable
- ✅ Even distribution (each backend gets equal traffic)
- ✅ No backend starvation

**Cons**:
- ❌ Doesn't consider backend load
- ❌ Slow backends get same traffic as fast ones
- ❌ Long requests accumulate on some backends

**Best for**: Backends with similar capacity, short-lived requests

---

##### Least Connections Balancer

```go
// LeastConnectionsBalancer sends traffic to backend with fewest active connections
type LeastConnectionsBalancer struct{}

func NewLeastConnectionsBalancer(backends []*Backend) *LeastConnectionsBalancer {
    return &LeastConnectionsBalancer{}
}

func (lc *LeastConnectionsBalancer) Next(backends []*Backend) *Backend {
    if len(backends) == 0 {
        return nil
    }
    
    // Find backend with minimum active connections
    minConnections := int32(^uint32(0) >> 1)  // Max int32
    var selected *Backend
    
    for _, backend := range backends {
        // Atomically read connection count
        connections := atomic.LoadInt32(&backend.ActiveConnections)
        
        if connections < minConnections {
            minConnections = connections
            selected = backend
        }
    }
    
    return selected
}
```

**Explanation**:
- Iterates through all backends
- Selects one with fewest active connections
- Uses atomic operations for thread safety

**How it works**:
```
Time 0s:
  Backend A: 5 connections
  Backend B: 2 connections ← Selected (least)
  Backend C: 8 connections
  → Request routed to B

Time 5s (after some requests):
  Backend A: 5 connections
  Backend B: 3 connections (previous request added 1)
  Backend C: 8 connections
  → Next request goes to A (now least)

Time 10s (long-running request started on A):
  Backend A: 15 connections (long requests piling up)
  Backend B: 3 connections ← Selected
  Backend C: 2 connections ← Or this one
  → Traffic shifts away from overloaded A
```

**Pros**:
- ✅ Adapts to backend load
- ✅ Long requests don't pile up on one server
- ✅ Better utilization of fast backends

**Cons**:
- ❌ Slightly more complex
- ❌ Requires tracking active connections
- ❌ Can cause "thundering herd" if all backends have 0 connections

**Best for**: Long-lived connections (WebSockets, streaming), varying request complexity

---

##### Random Balancer

```go
// RandomBalancer selects a random backend
type RandomBalancer struct{}

func NewRandomBalancer(backends []*Backend) *RandomBalancer {
    return &RandomBalancer{}
}

func (rb *RandomBalancer) Next(backends []*Backend) *Backend {
    if len(backends) == 0 {
        return nil
    }
    
    idx := rand.Intn(len(backends))
    return backends[idx]
}
```

**Explanation**:
- Truly random selection
- No state tracking needed
- Simple and fast

**How it works**:
```
Backends: [A, B, C]

Request 1: Random(0-2) = 1 → Backend B
Request 2: Random(0-2) = 2 → Backend C
Request 3: Random(0-2) = 1 → Backend B (can repeat)
Request 4: Random(0-2) = 0 → Backend A
Request 5: Random(0-2) = 2 → Backend C

Over 1000 requests:
  Backend A: ~333 requests (33.3%)
  Backend B: ~334 requests (33.4%)
  Backend C: ~333 requests (33.3%)
```

**Pros**:
- ✅ Extremely simple
- ✅ No synchronization needed (stateless)
- ✅ Good distribution with large request volumes
- ✅ No "hot backend" problem

**Cons**:
- ❌ Unpredictable (hard to debug)
- ❌ Can have short-term imbalances
- ❌ Doesn't consider backend load

**Best for**: Stateless applications, high traffic volume, caching layers

---

##### Weighted Balancer

```go
// WeightedBalancer distributes traffic according to backend weights
type WeightedBalancer struct {
    current uint32
}

func NewWeightedBalancer(backends []*Backend) *WeightedBalancer {
    return &WeightedBalancer{}
}

func (wb *WeightedBalancer) Next(backends []*Backend) *Backend {
    if len(backends) == 0 {
        return nil
    }
    
    // Calculate total weight
    totalWeight := 0
    for _, backend := range backends {
        totalWeight += backend.Weight
    }
    
    if totalWeight == 0 {
        // Fall back to round robin if no weights set
        idx := atomic.AddUint32(&wb.current, 1) % uint32(len(backends))
        return backends[idx]
    }
    
    // Select backend based on weight
    selection := atomic.AddUint32(&wb.current, 1) % uint32(totalWeight)
    
    currentWeight := uint32(0)
    for _, backend := range backends {
        currentWeight += uint32(backend.Weight)
        if selection < currentWeight {
            return backend
        }
    }
    
    return backends[0]  // Fallback
}
```

**Explanation**:
- Each backend has a weight (capacity indicator)
- Higher weight = more traffic
- Selection proportional to weight

**How it works**:
```
Backends:
  Backend A: Weight 100 (normal)
  Backend B: Weight 200 (2x capacity - newer/faster server)
  Backend C: Weight 50 (0.5x capacity - older server)
Total Weight: 350

Distribution:
  Backend A: 100/350 = 28.6% of traffic
  Backend B: 200/350 = 57.1% of traffic
  Backend C: 50/350 = 14.3% of traffic

Over 1000 requests:
  Backend A: ~286 requests
  Backend B: ~571 requests (2x of A)
  Backend C: ~143 requests (0.5x of A)
```

**Visual representation**:
```
Weight range: [0 ------------------------- 350]
Backend A:    [0 ----- 100]
Backend B:    [100 -------------- 300]
Backend C:    [300 ----- 350]

Random selection in range:
  50  → Falls in A's range → Backend A
  150 → Falls in B's range → Backend B
  320 → Falls in C's range → Backend C
```

**Pros**:
- ✅ Utilize more powerful backends efficiently
- ✅ Gradual rollout (new server gets low weight initially)
- ✅ Respect backend capacity differences

**Cons**:
- ❌ Requires manual weight configuration
- ❌ Doesn't adapt to real-time load
- ❌ More complex than round robin

**Best for**: Heterogeneous backends (different capacities), gradual deployments

---

### Comparison of Load Balancing Strategies

| Strategy | Best For | Pros | Cons | Complexity |
|----------|----------|------|------|------------|
| **Round Robin** | Similar backends, stateless apps | Simple, predictable | Ignores load | Low |
| **Least Connections** | Long requests, WebSockets | Load-aware, adaptive | Requires tracking | Medium |
| **Random** | High traffic, stateless | Simple, no state | Short-term imbalance | Low |
| **Weighted** | Different backend capacities | Respect capacity | Manual tuning | Medium |

**Real-world scenario - E-commerce site**:
```
Backend Group: "product-api"
  - Backend 1: 4 CPU, 8GB RAM → Weight 100
  - Backend 2: 8 CPU, 16GB RAM → Weight 200 (new server)
  - Backend 3: 2 CPU, 4GB RAM → Weight 50 (legacy)

Strategy: Weighted
Result: New server handles 2x traffic, legacy server handles 0.5x
```

---

## 📂 configs/ - Configuration Files

### `configs/proxy.yaml` - Main Configuration

**What it is**: The primary configuration file that defines how the proxy operates.

**Full example with explanations**:

```yaml
# Server configuration - defines where proxy listens
server:
  address: "0.0.0.0:443"           # Listen on all interfaces, port 443 (HTTPS)
  cert_file: "certs/server.crt"    # TLS certificate path
  key_file: "certs/server.key"     # TLS private key path
  
  # QUIC-specific settings
  max_idle_timeout: 30s             # Close idle connections after 30s
  max_incoming_streams: 100         # Max concurrent requests per connection
  keep_alive_period: 15s            # Send keep-alive PING every 15s

# Routing rules - map URL paths to backend groups
routes:
  # API routes
  - id: "api-v1"
    path: "/api/v1/*"                # Match all /api/v1/* requests
    methods: ["GET", "POST", "PUT", "DELETE"]  # Allow these HTTP methods
    backend_group: "api-servers"     # Send to api-servers backend group
    strip_prefix: "/api/v1"          # Remove /api/v1 before forwarding
    add_headers:
      X-Proxy-Version: "1.0"         # Add custom header
      
  # Static content routes
  - id: "static-files"
    path: "/static/*"
    methods: ["GET"]                 # Only GET allowed for static files
    backend_group: "static-servers"
    
  # Health check endpoint
  - id: "health"
    path: "/health"                  # Exact match only
    backend_group: "health-check"

# Backend server groups
backend_groups:
  # API server group with load balancing
  - id: "api-servers"
    strategy: "least_connections"    # Use least connections algorithm
    
    backends:
      - url: "http://10.0.0.1:8001"
        weight: 100
      - url: "http://10.0.0.2:8002"
        weight: 100
      - url: "http://10.0.0.3:8003"
        weight: 100
    
    # Active health checking
    health_check:
      enabled: true
      interval: 10s                  # Check every 10 seconds
      timeout: 5s                    # Timeout after 5 seconds
      path: "/health"                # Endpoint to check
      healthy_threshold: 2           # 2 successes = healthy
      unhealthy_threshold: 3         # 3 failures = unhealthy
      
      # Passive health checking
      passive:
        enabled: true
        max_failures: 5              # 5 consecutive errors = unhealthy
        observation_window: 60s      # Reset after 60s success
    
    # Connection pool settings
    connection_pool:
      max_idle_connections: 100
      max_connections_per_host: 10
      idle_timeout: 90s
    
    # Timeout settings
    timeout:
      connect: 5s                    # Time to establish connection
      request: 30s                   # Time to receive full response
      idle: 90s                      # Time before closing idle connection
    
    # Retry settings
    retry:
      max_attempts: 3                # Retry up to 3 times
      backoff: "exponential"         # 1s, 2s, 4s delays
      retry_on:                      # Retry on these conditions
        - "connection_error"
        - "timeout"
        - "5xx"                      # Server errors
  
  # Static file server group
  - id: "static-servers"
    strategy: "random"               # Random selection (simple, fast)
    
    backends:
      - url: "http://10.0.1.1:9000"
      - url: "http://10.0.1.2:9000"
    
    health_check:
      enabled: true
      interval: 30s                  # Less frequent (static files)
      path: "/ping"

# Telemetry configuration
telemetry:
  # Prometheus metrics
  metrics:
    enabled: true
    port: 9090                       # Metrics endpoint: :9090/metrics
    path: "/metrics"
  
  # OpenTelemetry tracing
  tracing:
    enabled: true
    endpoint: "localhost:4318"       # OTLP endpoint
    sample_rate: 1.0                 # 100% sampling (use 0.1 for 10% in prod)
  
  # Logging
  logging:
    level: "info"                    # debug, info, warn, error
    format: "json"                   # json or text
    output: "logs/proxy.log"         # Log file path
```

**Key concepts explained**:

**1. Why strip_prefix?**
```yaml
Client requests: GET /api/v1/users
strip_prefix: "/api/v1"
Backend receives: GET /users

Why? Backend doesn't know about /api/v1 versioning
       It just has /users endpoint
```

**2. Why add_headers?**
```yaml
add_headers:
  X-Forwarded-For: "${client_ip}"
  X-Request-ID: "${request_id}"

Backend needs:
  - Client IP for rate limiting
  - Request ID for distributed tracing
```

**3. Why health_check thresholds?**
```yaml
healthy_threshold: 2
unhealthy_threshold: 3

Prevents flapping:
  Success → Fail → Success → Fail (don't flip-flop status)
  Need consecutive failures to mark unhealthy
```

**4. Why connection pooling?**
```yaml
max_idle_connections: 100
max_connections_per_host: 10

Without pooling:
  Every request: Open TCP → TLS → Request → Close (150ms)

With pooling:
  First request: 150ms
  Subsequent: 50ms (reuse connection)
```

---

### `configs/example.yaml` - Minimal Configuration

**What it is**: Simplified config for quick testing.

```yaml
server:
  address: ":8443"
  cert_file: "certs/server.crt"
  key_file: "certs/server.key"

routes:
  - id: "default"
    path: "/*"
    backend_group: "backends"

backend_groups:
  - id: "backends"
    strategy: "round_robin"
    backends:
      - url: "http://localhost:8080"

telemetry:
  metrics:
    enabled: true
    port: 9090
  logging:
    level: "debug"
    format: "text"
```

**Use case**: Testing QUIC proxy locally without complex setup.

---

## 📂 certs/ - TLS Certificates

### Why QUIC Requires TLS

**Important**: QUIC **mandates TLS 1.3**. You cannot use QUIC without encryption.

**Reasoning**:
1. **Security**: All data encrypted by default
2. **Ossification prevention**: Middleboxes can't interfere with encrypted data
3. **0-RTT security**: TLS 1.3 enables faster resumption

### Certificate Structure

```
certs/
├── server.crt        # Public certificate (safe to share)
├── server.key        # Private key (NEVER share, add to .gitignore)
├── ca.crt            # Certificate Authority cert (for verification)
└── .gitkeep          # Keep directory in git
```

### Generating Development Certificates

**Using OpenSSL** (from Makefile):
```bash
openssl req -x509 -newkey rsa:4096 -keyout certs/server.key \
  -out certs/server.crt -days 365 -nodes \
  -subj "/CN=localhost"
```

**What each parameter means**:
- `-x509`: Create self-signed certificate
- `-newkey rsa:4096`: Generate 4096-bit RSA key
- `-keyout`: Where to save private key
- `-out`: Where to save certificate
- `-days 365`: Valid for 1 year
- `-nodes`: No password (for development)
- `-subj "/CN=localhost"`: Common Name = localhost

**For production**: Use Let's Encrypt or your organization's CA.

### Certificate Files Explained

#### `server.crt` - Public Certificate
```
-----BEGIN CERTIFICATE-----
MIIFazCCA1OgAwIBAgIUIAb4... (base64 encoded data)
-----END CERTIFICATE-----
```

**Contains**:
- Public key
- Domain name (Common Name)
- Issuer (who signed it)
- Validity period (not before / not after dates)
- Serial number

**Used for**:
- Client verifies proxy identity
- Establishes encrypted channel
- Ensures no man-in-the-middle

#### `server.key` - Private Key
```
-----BEGIN PRIVATE KEY-----
MIIJQgIBADANBgkqhkiG9w0B... (base64 encoded data)
-----END PRIVATE KEY-----
```

**Contains**:
- Private key corresponding to public key in certificate

**Security**: 
- ⚠️ **MUST be kept secret**
- Anyone with this key can impersonate your server
- Never commit to version control
- Set file permissions: `chmod 600 server.key`

---

## 📂 scripts/ - Automation Scripts

### `scripts/setup.sh` - Environment Setup

**What it does**: Prepares development environment.

```bash
#!/bin/bash
set -e  # Exit on any error

echo "Setting up QUIC Reverse Proxy development environment..."

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_VERSION="1.21"

if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    echo "Error: Go $REQUIRED_VERSION or higher required"
    exit 1
fi

# Install dependencies
echo "Installing Go dependencies..."
go mod download
go mod verify

# Create necessary directories
echo "Creating directories..."
mkdir -p build logs certs

# Generate certificates if they don't exist
if [ ! -f "certs/server.crt" ]; then
    echo "Generating TLS certificates..."
    make certs
fi

# Build the proxy
echo "Building proxy..."
make build

# Run tests
echo "Running tests..."
make test

echo "Setup complete! Run 'make run' to start the proxy."
```

**Explanation**:
- **set -e**: Stop script if any command fails
- **Version check**: Ensure Go 1.21+ installed
- **go mod download**: Download all dependencies
- **go mod verify**: Verify checksums (security)
- **mkdir -p**: Create directories (- p = no error if exists)
- **make certs**: Generate development certificates
- **make build**: Compile proxy binary
- **make test**: Run test suite

---

### `scripts/benchmark.sh` - Performance Testing

**What it does**: Measures proxy performance under load.

```bash
#!/bin/bash

echo "Starting QUIC Reverse Proxy Benchmark..."

# Start backend server
echo "Starting test backend..."
cd examples/node-backend
npm install
npm start &
BACKEND_PID=$!
sleep 2  # Wait for backend to start

# Start proxy
echo "Starting proxy..."
cd ../..
./build/proxy -config configs/test.yaml &
PROXY_PID=$!
sleep 2  # Wait for proxy to start

# Run benchmarks
echo "Running benchmarks..."

# Test 1: Throughput (requests per second)
echo "Test 1: Throughput"
ab -n 10000 -c 100 https://localhost:443/api/test

# Test 2: Latency distribution
echo "Test 2: Latency"
ab -n 1000 -c 10 -g latency.tsv https://localhost:443/api/test

# Test 3: Connection reuse
echo "Test 3: Connection reuse"
ab -n 5000 -c 50 -k https://localhost:443/api/test

# Cleanup
echo "Cleaning up..."
kill $PROXY_PID
kill $BACKEND_PID

echo "Benchmark complete! Check logs/proxy.log for metrics."
```

**Explanation**:
- **ab**: Apache Benchmark tool
  - `-n 10000`: Total 10,000 requests
  - `-c 100`: 100 concurrent connections
  - `-k`: Keep-alive (reuse connections)
  - `-g file`: Output latency data for graphing

**Expected output**:
```
Requests per second: 5000 [#/sec]
Time per request: 20.0 [ms] (mean)
Transfer rate: 1500 [Kbytes/sec]

Connection Times (ms)
              min  mean  median  max
Connect:        1    5      4    50
Processing:     5   15     12   100
Total:          6   20     16   150

Percentage of requests served within (ms)
  50%     16
  66%     20
  75%     25
  80%     30
  90%     40
  95%     60
  98%     80
  99%    100
 100%    150 (longest request)
```

---

Let me continue with the remaining sections...