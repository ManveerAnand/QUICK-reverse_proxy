# QUIC Reverse Proxy

An enterprise-grade, high-performance reverse proxy engineered with end-to-end QUIC and HTTP/3 support. It features comprehensive telemetry, dynamic load balancing, active health monitoring, and graceful failover capabilities designed for modern edge network infrastructure.

## Project Abstract

Modern network traffic demands low-latency, secure, and multiplexed connections. While many reverse proxies terminate QUIC/HTTP-3 at the edge to serve clients, they often revert to legacy HTTP/1.1 (TCP) when forwarding traffic to internal backend services. 

This project implements a true **end-to-end QUIC proxy**. It utilizes the `quic-go` implementation to terminate incoming TLS 1.3/HTTP-3 connections, dynamically select optimal transport streams, and natively proxy requests to backend microservices over HTTP/3, maintaining the performance benefits of QUIC throughout the entire request lifecycle.

---

## Core Architecture

The system is designed with a modular architecture, segregating the transport layer, data plane (routing and load balancing), and the control plane (health checks and telemetry).

### Transport Layer
*   **Ingress Protocol Support**: Natively terminates QUIC/HTTP-3 (UDP) alongside a standard HTTP/1.1 (TCP) fallback listener for legacy clients.
*   **Dynamic Egress Routing**: Employs a custom protocol-aware RoundTripper. Based on backend configuration, the proxy dynamically selects between `http3.RoundTripper` for QUIC-enabled microservices and `http.Transport` for standard HTTP/HTTPS backends.
*   **Zero-RTT & TLS 1.3**: Fully leverages TLS 1.3 to minimize connection overhead, supporting 0-RTT handshakes where applicable.

### Data Plane: Routing & Load Balancing
*   **Rule-based Routing**: Matches incoming requests against a prioritized list of rules evaluating paths, prefixes, host headers, and HTTP methods.
*   **Algorithms**: Distributes traffic across backend pools using configurable strategies:
    *   `round_robin`: Sequential distribution.
    *   `least_connections`: Routes to the backend with the fewest active, in-flight requests.
    *   `weighted`: Proportionally allocates traffic based on assigned backend weights.

### Control Plane: Health Monitoring
*   **Active Probing**: Executes scheduled background HTTP probes against backend endpoints.
*   **Automated Failover**: Isolates and removes degraded backends from the active load-balancer rotation upon breaching failure thresholds.
*   **Self-Healing**: Automatically reinstates backends into the rotation once consecutive successful probes meet the recovery threshold.

---

## Comprehensive Telemetry Stack

Visibility is treated as a first-class citizen. The proxy is instrumented to export data across the three pillars of observability:

1.  **Metrics (Prometheus)**: Exposes granular metrics at the `/metrics` endpoint, capturing total requests, active connections, status code distributions, payload sizes, and precise latency histograms for both client-to-proxy and proxy-to-backend hops.
2.  **Distributed Tracing (OpenTelemetry)**: Injects and propagates OpenTelemetry span contexts, exporting distributed trace data to Jaeger for complex request lifecycle analysis.
3.  **Structured Logging**: Emits JSON-formatted logs suitable for aggregation (e.g., ELK or Splunk), detailing request outcomes, health check state transitions, and transport-level errors.

*A pre-configured Grafana dashboard is provided in the repository to visualize this data immediately upon deployment.*

---

## Getting Started

### Prerequisites
*   Go 1.24 or later
*   OpenSSL (for local certificate generation)
*   Docker and Docker Compose

### Local Development Environment

The repository includes a complete `docker-compose` environment that spins up the proxy, a sample HTTP/3 backend service, and the full observability stack (Prometheus, Grafana, and Jaeger).

1.  **Initialize the Environment**
    This command downloads Go modules and generates the self-signed certificates required for local TLS 1.3 termination.
    ```bash
    make init-project
    ```

2.  **Launch the Infrastructure**
    ```bash
    make docker-compose-up
    ```

3.  **Verify End-to-End QUIC Proxying**
    Ensure you have an HTTP/3 capable client (like a recent version of `curl` compiled with HTTP/3 support).
    ```bash
    # The request hits the proxy via HTTP/3 and is forwarded to the backend via HTTP/3
    curl --http3 https://localhost:443/api/status -k
    ```

### Component Access Points
*   **QUIC Proxy Ingress**: `https://localhost:443`
*   **Prometheus**: `http://localhost:9091`
*   **Grafana Dashboards**: `http://localhost:3001` *(Default credentials: admin/admin)*
*   **Jaeger UI**: `http://localhost:16686`

---

## Configuration Reference

The proxy behavior is dictated by a YAML configuration file.

### Backend Configuration Schema
The backend definition allows precise control over protocol selection and health monitoring.

```yaml
backends:
  - name: "primary-http3-service"
    targets:
      - "localhost:8081"
    # Determines the egress transport. Options: "h3", "https", "http".
    protocol: "h3" 
    # Bypasses TLS validation for self-signed backend certificates.
    tls_skip_verify: true 
    weight: 1
    health_check:
      enabled: true
      interval: "10s"
      timeout: "2s"
      path: "/health"
```

### Full Configuration Example
Refer to `configs/example.yaml` in the repository for a complete example covering server TLS parameters, routing rules, and telemetry integrations.

---

## Development Guide

### Build Targets

```bash
# Compile for the host architecture
make build

# Cross-compile binaries for Linux, macOS, and Windows
make build-all

# Run the proxy locally with live-reloading (requires 'air')
make dev
```

### Quality Assurance

```bash
# Execute unit and integration tests
make test

# Generate an HTML test coverage report
make test-coverage

# Run static analysis and linting
make lint
```

## License

This project is licensed under the MIT License. See the LICENSE file for full details.