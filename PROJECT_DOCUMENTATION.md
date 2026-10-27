# QUIC Reverse Proxy: A Detailed Explanation

## 1. Executive Summary

The **QUIC Reverse Proxy** is a high-performance, production-ready reverse proxy server written in Go. Its primary feature is the use of the **QUIC** protocol (and HTTP/3) to accelerate traffic between clients and the proxy, offering lower latency and better performance than traditional TCP-based proxies.

It is designed for modern, cloud-native environments and comes with a comprehensive suite of features out-of-the-box, including:

- **High-Performance Proxying:** Built on QUIC/HTTP-3 for speed, with a fallback to standard HTTP/1.1 for compatibility.
- **Intelligent Routing:** Advanced, path-based routing to direct traffic to different backend services.
- **Load Balancing:** Distributes incoming traffic across multiple backend servers using various strategies to ensure high availability and reliability.
- **Health Monitoring:** Actively monitors the health of backend services and automatically routes traffic away from unhealthy ones.
- **Full-Stack Observability:** Provides deep insights into the proxy's performance and behavior through:
    - **Metrics** with Prometheus.
    - **Distributed Tracing** with Jaeger.
    - **Structured Logging**.
- **Containerization:** Comes with ready-to-use Docker and Kubernetes configurations for easy deployment.

In essence, this project is not just a proxy; it's a complete traffic management and observability solution for microservices and other distributed systems.

---

## 2. Core Concepts Explained

To understand the project, let's break down the key terms and technologies.

### a. What is QUIC and HTTP/3?

- **QUIC (Quick UDP Internet Connections)** is a modern transport protocol developed by Google, designed to be faster and more efficient than TCP, which has been the standard for decades. It runs over UDP (User Datagram Protocol).
- **Key Advantages:**
    - **Reduced Latency:** QUIC establishes connections much faster. A new connection can be made in a single round-trip, and subsequent connections to a known server can have **Zero Round-Trip Time (0-RTT)**, making it feel almost instantaneous.
    - **No Head-of-Line Blocking:** In traditional HTTP/2 over TCP, if one data packet is lost, all other data streams have to wait for it to be retransmitted, blocking everything. QUIC is multiplexed, meaning streams are independent; a lost packet only affects its own stream, not others.
    - **Connection Migration:** If you switch from Wi-Fi to cellular data, your TCP connection breaks. QUIC uses a unique connection ID, allowing your device to switch networks without dropping the connection.
- **HTTP/3** is the third major version of the HTTP protocol, which runs over QUIC instead of TCP. This project uses HTTP/3 to serve traffic, bringing all the benefits of QUIC to web requests.

### b. What is a Reverse Proxy?

A reverse proxy is a server that sits in front of one or more backend web servers. Instead of a client connecting directly to a backend service, it connects to the reverse proxy. The proxy then forwards the client's request to the appropriate backend.

- **Why use it?**
    - **Load Balancing:** Distribute traffic across many servers so no single server gets overloaded.
    - **Security:** Hide the identity and characteristics of your backend servers.
    - **TLS/SSL Termination:** The proxy can handle encrypting and decrypting traffic, offloading that work from the backend servers.
    - **Caching:** Store copies of responses to speed up future requests.
    - **Centralized Management:** Apply security policies, logging, and monitoring in one place.

This project acts as a reverse proxy, but a highly advanced one that uses QUIC.

### c. Load Balancing Algorithms

This project distributes traffic to backend servers using several algorithms defined in `internal/proxy/loadbalancer.go`:

- **Round Robin (Default):** Requests are sent to servers in a simple, rotating sequence (Server 1, then Server 2, then Server 3, and back to 1). It's simple and effective for evenly distributing traffic.
- **Least Connections:** The proxy sends the next request to the backend server that currently has the fewest active connections. This is smart for handling requests that may take a variable amount of time to process.
- **Weighted:** You can assign a "weight" to each backend server. A server with a weight of 2 will receive twice as much traffic as a server with a weight of 1. This is useful if your servers have different capacities.

### d. Health Checking

The proxy doesn't blindly forward traffic. The `pkg/health/checker.go` component continuously sends small "ping" requests to a health endpoint on each backend server (e.g., `/health`).

- If a backend responds successfully, it's marked **"healthy."**
- If it fails to respond multiple times (the `unhealthy_threshold` in `proxy.yaml`), it is marked **"unhealthy,"** and the load balancer will stop sending traffic to it.
- Once an unhealthy server starts responding again, it is marked "healthy" and brought back into the load balancing pool.

This prevents users from experiencing errors due to a failing backend server.

### e. The Three Pillars of Observability

This project provides a complete observability stack, allowing you to understand its internal state and behavior in detail.

1.  **Metrics (The "What") with Prometheus:**
    -   **What it is:** Metrics are numerical measurements of the proxy's activity over time. The proxy exposes these at the `/metrics` endpoint, as seen in `internal/telemetry/metrics.go`.
    -   **Examples:**
        -   `http_requests_total`: How many requests have been processed?
        -   `http_request_duration_seconds`: How long are requests taking?
        -   `go_goroutines`: How much concurrent work is the proxy doing?
        -   `backend_health_status`: Is a backend healthy or not?
    -   **Prometheus** is a tool (defined in `docker-compose.yml`) that scrapes (collects) these metrics every few seconds and stores them. **Grafana** is then used to create beautiful dashboards from this data, like the one in `monitoring/grafana/dashboards/quic-proxy-dashboard.json`.

2.  **Distributed Tracing (The "Where") with Jaeger:**
    -   **What it is:** In a microservices world, a single user request might travel through multiple services. Tracing follows the entire journey of that request, showing you how long it spent in each service. This is invaluable for pinpointing bottlenecks.
    -   The `internal/telemetry/tracing.go` file sets up the Jaeger tracer. When a request comes in, the proxy starts a "span" (a unit of work) and passes a trace ID to the backend, which can then add its own spans.
    -   **Jaeger** is the tool (defined in `docker-compose.yml`) that collects and visualizes these traces, showing you a timeline of the request's journey.

3.  **Structured Logging (The "Why"):
    -   **What it is:** Instead of plain text logs, structured logging writes logs in a machine-readable format like JSON. Each log entry has key-value pairs (e.g., `"method": "GET"`, `"status": 200`).
    -   As seen in `cmd/proxy/main.go`, the project uses the `logrus` library configured with a `JSONFormatter`.
    -   **Why it's better:** You can easily filter, search, and analyze logs (e.g., "show me all logs for requests with a 500 error status").

---

## 3. Project Architecture & Request Flow

The architecture is best understood by following a request from the client to the backend and back. The `docker-compose.yml` file defines all the moving parts.

```
 Client (HTTP/3) → QUIC Proxy (Port 443/udp) → Load Balancer ↘
                                                                Backend Services
 Client (HTTP/1.1) → HTTP Fallback (Port 80/tcp) ↗           (backend1, backend2)
                                                                ↑
                                                          Health Checker
      ↓
 Telemetry Stack
 ├── Prometheus (Metrics)
 ├── Grafana (Dashboards)
 └── Jaeger (Tracing)
```

**Step-by-Step Request Flow:**

1.  A **Client** sends a request. Let's say it's `GET /api/users`.
2.  The **QUIC Reverse Proxy** receives the request on its QUIC port (443/udp).
3.  The **Router** (`internal/proxy/router.go`) immediately gets to work. It checks the request against the `routing.rules` in `configs/proxy.yaml`.
    -   It finds a rule: `path: "/api/*"` matches `/api/users`.
    -   This rule points to the backend named `"api-service"`.
4.  The request is handed to the **Load Balancer** (`internal/proxy/loadbalancer.go`).
    -   It looks up the `"api-service"` backend group, which contains two targets: `http://backend1:80` and `http://backend2:3000`.
    -   The load balancing algorithm for this group is `round_robin`. Assuming the last request went to `backend1`, this one will go to `backend2`.
5.  The **Health Checker** (`pkg/health/checker.go`) has already been running in the background. The load balancer confirms that `backend2` is currently marked as healthy.
6.  The **Proxy Handler** (`internal/proxy/handler.go`) forwards the request to `http://backend2.com:3000/api/users`.
    -   During this process, the **Telemetry Manager** (`internal/telemetry/manager.go`) is active:
        -   A **trace** is started with Jaeger to time the entire operation.
        -   **Metrics** are recorded: `http_requests_total` is incremented, and the request duration is measured.
7.  The **Node.js Backend Service** (`backend2`) receives the request, processes it, and returns a JSON response.
8.  The proxy receives the response from the backend and forwards it back to the original client.
9.  The trace is completed, and the final metrics (like response size and status code) are recorded.

---

## 4. Configuration (`configs/proxy.yaml`)

This file is the brain of the proxy. It tells it how to behave.

-   `server`: Configures the main proxy server, including the listening `address` (`:443`), TLS certificate files (`cert_file`, `key_file`), and QUIC-specific parameters like `idle_timeout`.
-   `backends`: An array defining the groups of backend services the proxy can send traffic to.
    -   Each backend has a `name` (e.g., `api-service`), a list of `targets` (the actual server URLs), and its own `load_balancer` algorithm.
    -   It also defines the `health_check` behavior for that group (path, interval, etc.).
-   `routing`: This is where the magic happens. It contains the `rules` that map incoming requests to backends.
    -   Rules are processed by `priority` (higher numbers first).
    -   A rule can match based on `path` (with wildcards like `*`), HTTP `methods`, and more.
    -   `strip_prefix: true` is a useful feature that removes the matched part of the path before forwarding (e.g., a request to `/web/home` with `strip_prefix: true` on the `/web/*` rule would be forwarded to the backend as just `/home`).
-   `telemetry`: Configures the observability stack.
    -   `metrics`: Enables Prometheus, sets its port and path.
    -   `tracing`: Enables Jaeger, sets its endpoint URL, and the `sample_rate` (e.g., `0.1` means trace 10% of requests).
    -   `logging`: Sets the log `level` (info, debug) and `format` (json, text).

---

## 5. Code Structure

The Go code is well-organized by function:

-   `cmd/proxy/main.go`: The entry point of the application. It parses flags, loads the configuration, initializes telemetry, and starts the proxy server.
-   `internal/config/`: Handles loading, parsing, and validating the `proxy.yaml` file. The `types.go` file defines the Go structs that map to the YAML structure.
-   `internal/proxy/`: The core logic.
    -   `server.go`: The main server struct that ties everything together.
    -   `handler.go`: The HTTP handler that receives requests and uses the router and load balancer to process them.
    -   `router.go`: Implements the routing logic based on the configuration rules.
    -   `loadbalancer.go`: Implements the backend selection algorithms and manages the list of healthy backends.
-   `internal/quic/`: Contains the low-level QUIC server (`server.go`) and client (`client.go`) implementations, built on the `quic-go` library.
-   `internal/telemetry/`:
    -   `manager.go`: A coordinator for all telemetry components.
    -   `metrics.go`: Defines and registers all the Prometheus metrics.
    -   `tracing.go`: Sets up the OpenTelemetry and Jaeger tracer.
-   `pkg/health/checker.go`: A reusable package for performing health checks. It's in `pkg` because it's generic and could be used in other projects.

This clear separation of concerns makes the project easy to understand, maintain, and extend.
