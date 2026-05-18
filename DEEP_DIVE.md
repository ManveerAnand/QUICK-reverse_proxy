# QUIC Reverse Proxy: Architecture Deep Dive

This document provides a highly detailed technical breakdown of the QUIC Reverse Proxy. It is intended for systems engineers, network architects, and core maintainers who need to understand the intricate inner workings, performance boundaries, and scalability models of this edge gateway.

---

## 1. Transport Layer Internals

The proxy is built to bridge the gap between emerging web transport standards and internal microservice architectures. The critical innovation in this proxy is its avoidance of "QUIC termination," opting instead for a true end to end QUIC circuit.

### Ingress: The UDP Socket and quic-go Model
Unlike a standard HTTP/1.1 reverse proxy that listens on a single `net.TCPListener` and forks goroutines for `Accept()`, this proxy binds to a `net.UDPConn`. 
* **Packet Processing:** The `quic-go` engine reads raw UDP packets. Because QUIC multiplexes multiple logical connections over a single 5 tuple (Source IP, Source Port, Dest IP, Dest Port), the engine parses the QUIC header in user space to extract the **Connection ID (CID)**.
* **Threading Model:** The engine dispatches packets to individual connection state machines. Each QUIC connection maintains its own crypto context (TLS 1.3 keys) and congestion window (e.g., Cubic or BBR). This user space multiplexing creates high CPU overhead compared to kernel space TCP.
* **0 RTT Handshakes:** By storing session tickets, the proxy allows returning clients to send HTTP/3 request data in their very first flight of packets, eliminating the 1 RTT TLS handshake penalty.

### Egress: Dynamic RoundTripper Provisioning
When the proxy needs to forward a request to an upstream service, it must bridge the internal `httputil.ReverseProxy` with the network.
* **The `http.RoundTripper` Interface:** The proxy implements a dynamic transport factory. If the backend is marked as `protocol: h3`, the proxy instantiates an `http3.RoundTripper`.
* **Multiplexing Preservation:** A standard TCP proxy suffers from Head of Line (HoL) blocking. If packet 2 is lost, packet 3 cannot be processed. By routing upstream over HTTP/3, the proxy opens independent QUIC streams for each concurrent request to the backend. A dropped packet on Stream A does not stall Stream B, preserving ultra low latency for parallel microservice calls.

---

## 2. Data Plane: Routing and State Management

The data plane is strictly optimized for the hot path. Every nanosecond spent in the router or load balancer degrades proxy throughput.

### RCU (Read Copy Update) State Pattern
The `LoadBalancer` must maintain the health status of upstream backends. In a naive implementation, a global `sync.RWMutex` would guard the backend list.
* **The Contention Problem:** At 50,000 concurrent requests, thousands of reader goroutines constantly locking the RWMutex cause "cache line bouncing" across CPU cores, completely destroying L1/L2 cache efficiency.
* **The Solution:** The proxy relies heavily on atomic operations. Active connections are tracked using `atomic.AddInt32`. Health state is tracked via `atomic.LoadInt32`. During routing, the handler reads these atomic integers lock free. The only time locks are acquired is during a complete backend configuration reload, which occurs completely out of band of the request cycle.

### Algorithmic Load Distribution
The proxy implements several algorithms, carefully optimized to avoid blocking:
1. **Round Robin:** Instead of a blocking mutex to track the index, it uses `atomic.AddInt32` modulo the backend count.
2. **Least Connections:** Iterates through the backend pool reading the atomic connection counter. While technically an O(n) operation, the small size of internal backend pools makes this negligible compared to the network I/O overhead.

---

## 3. The Control Plane: Active Probing

The Control Plane operates asynchronously to the Data Plane. Its primary responsibility is maintaining the integrity of the routing pool.

### Health Check Goroutines
* Upon initialization, the `LoadBalancer` forks a lightweight goroutine for every configured backend target.
* These goroutines sleep on an OS timer (`time.Ticker`). When triggered, they execute an isolated HTTP GET request to the backend's `/health` endpoint.
* **Threshold State Machine:** A backend does not immediately toggle from healthy to unhealthy upon a single failure (which could be a transient network blip). It maintains a local counter. Only when the `unhealthy_threshold` (e.g., 3 consecutive failures) is breached does it execute the atomic write to mark the backend as offline. 

---

## 4. Performance Bottlenecks and Mitigation

For a Go based QUIC proxy, bottlenecks differ drastically from standard Nginx/Envoy deployments. 

### Bottleneck 1: User Space UDP CPU Exhaustion
Because the Linux kernel does not natively process QUIC, `quic-go` relies heavily on user space processing.
* **Mitigation:** The system relies on the underlying OS supporting **UDP GRO (Generic Receive Offload)** and **GSO (Generic Segmentation Offload)**. These features allow the NIC and OS to batch multiple UDP payloads into a single buffer before waking up the Go runtime, drastically reducing syscall overhead.
* **Future Evolution:** For extreme scale, the ingress pipeline could be rewritten using **eBPF/XDP** (Express Data Path) to parse QUIC headers directly in the NIC driver, bypassing the Linux network stack entirely.

### Bottleneck 2: Garbage Collection (GC) Pressure
A reverse proxy is fundamentally a data copy machine. Reading from the ingress socket and writing to the egress socket requires byte buffers.
* **Mitigation:** The architecture mandates strict memory management. The hot path utilizes `sync.Pool` to recycle `[]byte` buffers. By reusing memory instead of allocating new slices per request, the system minimizes the frequency of Go's Stop the World (STW) garbage collection pauses, which directly impact p99 latency tails.

---

## 5. Distributed Scalability Strategy

The proxy itself is entirely stateless (excluding the ephemeral TLS session tickets). This allows for massive horizontal scale.

### Layer 4 Scaling and Connection Migration
To handle millions of connections, multiple proxy nodes must be placed behind a Layer 4 (UDP) Load Balancer (e.g., HAProxy, Maglev).
* **The QUIC Challenge:** QUIC supports Connection Migration. A mobile client might switch from a WiFi IP to a Cellular IP mid request. The QUIC connection remains valid because it is identified by the Connection ID (CID), not the IP 5 tuple.
* **The L4 Requirement:** Standard IP hash load balancing will break Connection Migration. The upstream Layer 4 load balancer *must* be capable of deeply inspecting the QUIC packet, extracting the CID, and consistently hashing that CID to the correct backend proxy instance. If a packet is routed to the wrong proxy instance, that instance will not have the corresponding TLS state, and the packet will be dropped.

---

## 6. Telemetry and Distributed Tracing Architecture

The proxy treats observability as a core requirement, integrating heavily with Prometheus and OpenTelemetry.

### Asynchronous Metrics
Prometheus metrics (e.g., `http_request_duration_seconds`) are recorded at the very end of the `Handler` execution. To prevent metric recording from adding latency to the response time, the `telemetry` package relies on highly optimized concurrent data structures provided by `client_golang`.

### Context Propagation across the QUIC Boundary
Standard HTTP proxies propagate trace contexts (e.g., `traceparent` headers) via standard HTTP headers. Because this proxy routes over HTTP/3, these headers are compressed using **QPACK** (QUIC's equivalent of HPACK). The `http3.RoundTripper` handles the QPACK compression/decompression seamlessly, ensuring that Jaeger trace spans remain perfectly contiguous from the client, through the proxy, and down into the internal microservices, regardless of the transport protocol changes.
