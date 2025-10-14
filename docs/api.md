# API Documentation for QUIC Reverse Proxy

## Overview

This document provides an overview of the API endpoints available in the QUIC Reverse Proxy application. The reverse proxy supports QUIC and HTTP/3 protocols, enabling efficient and secure communication between clients and backend services. 

## Base URL

The base URL for the API is determined by the server configuration in `proxy.yaml`. For example, if the server is running on `localhost` with port `443`, the base URL would be:

```
https://localhost:443
```

## Endpoints

### 1. Health Check

- **Endpoint:** `/health`
- **Method:** `GET`
- **Description:** Checks the health status of the reverse proxy service.
- **Response:**
  - **200 OK:** Service is healthy.
  - **503 Service Unavailable:** Service is not healthy.

#### Example Request

```
GET /health HTTP/3
```

#### Example Response

```json
{
  "status": "healthy"
}
```

### 2. Proxy Request

- **Endpoint:** `/{path}`
- **Method:** `ANY`
- **Description:** Forwards incoming requests to the appropriate backend service based on the routing configuration.
- **Request Parameters:**
  - `path`: The path of the request that will be forwarded to the backend service.
  
- **Response:**
  - **200 OK:** Successful response from the backend service.
  - **4xx Client Errors:** Errors related to the request (e.g., 404 Not Found).
  - **5xx Server Errors:** Errors related to the backend service.

#### Example Request

```
GET /api/v1/resource HTTP/3
```

#### Example Response

```json
{
  "data": {
    "id": 1,
    "name": "Resource Name"
  }
}
```

### 3. Telemetry Metrics

- **Endpoint:** `/metrics`
- **Method:** `GET`
- **Description:** Exposes metrics for observability, compatible with Prometheus.
- **Response:**
  - **200 OK:** Returns metrics in Prometheus format.

#### Example Request

```
GET /metrics HTTP/3
```

#### Example Response

```
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET", handler="/api/v1/resource"} 1023
```

## Usage Examples

### Making a Proxy Request

To make a request through the reverse proxy, simply send a request to the desired path. The reverse proxy will handle routing to the appropriate backend service.

```bash
curl -X GET https://localhost:443/api/v1/resource
```

### Checking Health Status

To check the health of the reverse proxy, you can use the following command:

```bash
curl -X GET https://localhost:443/health
```

### Accessing Metrics

To access the telemetry metrics, use:

```bash
curl -X GET https://localhost:443/metrics
```

## Conclusion

This API documentation outlines the key endpoints available in the QUIC Reverse Proxy application. For further details on configuration and deployment, please refer to the `README.md` and configuration files.