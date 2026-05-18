# QUIC Reverse Proxy: Architecture Deep Dive

This document provides a comprehensive technical breakdown of the QUIC Reverse Proxy. It is intended for systems engineers and architects who wish to understand the inner workings, performance characteristics, scaling strategies, and design philosophy behind this edge gateway.

---

## 1. System Architecture and Request Lifecycle

The proxy acts as an asynchronous, non-blocking gateway that terminates incoming connections, evaluates routing logic, and streams traffic to downstream microservices.

### The "End-to-End" QUIC Pipeline
The most distinguishing feature of this proxy is its capability to maintain a full QUIC circuit. 
1. **Ingress (Edge Termination)**: The proxy utilizes the `quic-go` implementation to terminate incoming UDP streams. This enforces TLS 1.3 and handles the complex state machines required for QUIC's 0-RTT handshakes and congestion control.
2. **Dynamic Transport Layer**: Once the request hits the unified `Handler`, the proxy evaluates the target backend's `Protocol` configuration.
3. **Egress (Backend Proxying)**: Instead of naively falling back to TCP for upstream communication, the proxy provisions an `http3.RoundTripper`. This ensures that traffic directed at QUIC-enabled microservices flows entirely over UDP, completely avoiding traditional TCP Head-of-Line blocking within the internal network.

### The Control Plane
Running asynchronously alongside the data plane is the control plane, primarily responsible for **Active Health Checking**.
* The `LoadBalancer` spins up isolated goroutines for each configured backend.
* These goroutines execute periodic, timeout-bound probes. 
* State mutations (e.g., a backend transitioning from `healthy` to `unhealthy`) are managed via an RCU (Read-Copy-Update) or Mutex-guarded state pattern, ensuring that the critical path for routing incoming requests remains extremely fast and never blocks on a network timeout.

---

## 2. Performance Characteristics and Bottlenecks

Understanding the limitations of any system is critical for deployment at scale. For a Go-based QUIC proxy, the bottlenecks fundamentally shift away from traditional OS-level socket exhaustion toward CPU and Memory bounds.

### User-Space UDP Processing (The CPU Bound)
Unlike TCP, which benefits from decades of optimization within the Linux kernel (including hardware offloading in NICs), QUIC is largely processed in user-space.
* **The Challenge**: Processing millions of UDP packets requires constant system calls and context switching between kernel space and user space. This results in the CPU becoming the primary bottleneck long before raw network bandwidth is saturated.
* **The Solution**: Modern deployments mitigate this by ensuring kernel features like **UDP GRO (Generic Receive Offload)** and **GSO (Generic Segmentation Offload)** are enabled. These features allow the OS to batch UDP packets, drastically reducing the number of system calls the Go runtime must make.

### Garbage Collection Pressure (The Memory Bound)
As a reverse proxy, the system must copy massive volumes of data from the ingress socket to the egress socket.
* **The Challenge**: If every incoming request allocates new memory for headers, byte buffers, and telemetry context, the Go Garbage Collector (GC) will be forced to run constantly, leading to latency spikes (GC pauses).
* **The Solution**: The proxy mitigates allocation overhead by leveraging `sync.Pool`. By retaining and recycling byte buffers after a request completes, the system approaches a "zero-allocation" critical path, smoothing out latency tails at high percentiles.

### State Synchronization
The `LoadBalancer` must maintain the state of backend health and active connections to execute algorithms like `least_connections`.
* **The Challenge**: High concurrency against a single `sync.RWMutex` can cause cache-line bouncing across CPU cores.
* **The Solution**: The proxy employs atomic operations (`atomic.LoadInt32`, `atomic.AddInt32`) for counting active connections and reading health state, entirely bypassing the need for locking during the hot path of request routing.

---

## 3. Scalability Strategies

The proxy is architected to be completely stateless regarding client sessions (beyond the TLS state managed by `quic-go`). This allows for extensive scalability.

### Vertical Scaling (Scaling Up)
Because the proxy heavily utilizes Go's goroutines for extreme concurrency, it scales almost linearly with additional CPU cores. The primary ceiling for vertical scaling is the physical bandwidth of the network interface and the efficiency of the OS's UDP stack.

### Horizontal Scaling (Scaling Out)
To scale beyond a single node, multiple proxy instances can be deployed behind a Layer 4 Load Balancer (e.g., HAProxy, Maglev, or cloud-native network load balancers).

**The Connection Migration Challenge**: 
A key feature of QUIC is *Connection Migration*—the ability for a client to change its IP address (e.g., switching from WiFi to Cellular) without dropping the active connection. 
* To support this in a horizontally scaled environment, the upstream Layer 4 Load Balancer must be configured to route UDP packets based on the **QUIC Connection ID (CID)** rather than the traditional IP 5-tuple. 
* If CID-based routing is not implemented at the edge, a migrating client's packets might be sent to Proxy Instance B, which does not hold the TLS session keys established on Proxy Instance A, resulting in a dropped connection.

---

## 4. Telemetry and Observability

To operate a distributed system safely, deep visibility is required. The proxy integrates a comprehensive observability stack:

* **Granular Metrics**: The system tracks `http_request_duration_seconds` (latency histograms), `backend_requests_total`, and active connections. By analyzing these metrics, operators can pinpoint whether latency originates from the client-to-proxy connection or the proxy-to-backend connection.
* **Trace Propagation**: OpenTelemetry extracts incoming trace headers and propagates them to backend services. This ensures that a single request can be visualized across the entire microservice architecture, allowing for precise bottleneck identification in complex transactions.
