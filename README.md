<div align="center">

# QUIC Reverse Proxy
**Enterprise grade, high performance edge gateway with native HTTP/3 support**

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=for-the-badge&logo=go)](https://golang.org)
[![Protocol](https://img.shields.io/badge/Protocol-QUIC%20%7C%20HTTP%2F3-blueviolet?style=for-the-badge)](#)
[![Docker Support](https://img.shields.io/badge/Docker-Supported-2496ED?style=for-the-badge&logo=docker)](#)
[![License](https://img.shields.io/badge/License-MIT-success?style=for-the-badge)](#)

</div>

<br/>

Modern network traffic demands low latency, secure, and multiplexed connections. While many reverse proxies terminate QUIC at the edge, they often revert to legacy TCP when forwarding traffic to internal backend services. 

This project implements a true **end to end QUIC proxy**. It terminates incoming TLS 1.3 connections and natively proxies requests to backend microservices over HTTP/3, maintaining the performance benefits of QUIC throughout the entire request lifecycle.

<br/>

## Core Architecture

The system is designed with a modular architecture, segregating the transport layer, data plane, and the control plane.

### Transport Layer
* **Native Ingress** <br/>
Natively terminates QUIC UDP alongside a standard TCP fallback listener for legacy clients.
* **Dynamic Egress** <br/>
Employs a custom protocol aware RoundTripper. Based on backend configuration, the proxy dynamically selects between HTTP/3 for QUIC enabled microservices and standard HTTP for legacy backends.
* **Zero RTT & TLS 1.3** <br/>
Fully leverages TLS 1.3 to minimize connection overhead, supporting zero RTT handshakes where applicable.

### Data Plane
* **Rule Based Routing** <br/>
Matches incoming requests against a prioritized list of rules evaluating paths, prefixes, host headers, and HTTP methods.
* **Dynamic Distribution** <br/>
Distributes traffic across backend pools using configurable strategies including round robin, least connections, and weighted distribution.

### Control Plane
* **Active Probing** <br/>
Executes scheduled background HTTP probes against backend endpoints.
* **Automated Failover** <br/>
Isolates and removes degraded backends from the active load balancer rotation upon breaching failure thresholds.
* **Self Healing** <br/>
Automatically reinstates backends into the rotation once consecutive successful probes meet the recovery threshold.

<br/>

## Comprehensive Telemetry

Visibility is treated as a first class citizen. The proxy is instrumented to export data across the three pillars of observability.

> **Metrics (Prometheus)**  
> Exposes granular metrics capturing total requests, active connections, status code distributions, payload sizes, and precise latency histograms.

> **Distributed Tracing (OpenTelemetry)**  
> Injects and propagates OpenTelemetry span contexts, exporting distributed trace data to Jaeger for complex request lifecycle analysis.

> **Structured Logging**  
> Emits JSON formatted logs suitable for aggregation, detailing request outcomes, health check state transitions, and transport level errors.

<br/>

## Getting Started

### Local Development Environment

The repository includes a complete Docker environment that spins up the proxy, a sample HTTP/3 backend service, and the full observability stack (Prometheus, Grafana, and Jaeger).

**1. Initialize the Environment**  
This command downloads Go modules and generates the self signed certificates required for local TLS 1.3 termination.
```bash
make init-project
```

**2. Launch the Infrastructure**  
```bash
make docker-compose-up
```

**3. Verify QUIC Proxying**  
Ensure you have an HTTP/3 capable client.
```bash
curl --http3 https://localhost:443/api/status -k
```

<br/>

## Configuration Reference

The proxy behavior is dictated by a YAML configuration file.

### Backend Configuration Schema
The backend definition allows precise control over protocol selection and health monitoring.

```yaml
backends:
  - name: "primary_service"
    targets:
      - "localhost:8081"
    protocol: "h3" 
    tls_skip_verify: true 
    weight: 1
    health_check:
      enabled: true
      interval: "10s"
      timeout: "2s"
      path: "/health"
```

<br/>

## Development Guide

**Build Targets**
```bash
make build       # Compile for the host architecture
make build-all   # Cross compile binaries
make dev         # Run with live reloading
```

**Quality Assurance**
```bash
make test          # Execute unit and integration tests
make test-coverage # Generate an HTML test coverage report
make lint          # Run static analysis and linting
```

<br/>

## License

Built by [Manveer Anand](https://github.com/ManveerAnand).  
This project is licensed under the MIT License. See the LICENSE file for full details.