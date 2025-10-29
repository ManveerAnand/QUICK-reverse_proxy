package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/os-dev/quic-reverse-proxy/internal/telemetry"
	"github.com/sirupsen/logrus"
)

// Handler handles HTTP requests and forwards them to backend services
type Handler struct {
	router       *Router
	loadBalancer *LoadBalancer
	metrics      *telemetry.Metrics
}

// NewHandler creates a new proxy handler
func NewHandler(router *Router, loadBalancer *LoadBalancer, metrics *telemetry.Metrics) *Handler {
	return &Handler{
		router:       router,
		loadBalancer: loadBalancer,
		metrics:      metrics,
	}
}

// ServeHTTP implements the http.Handler interface
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Handle health check endpoint
	if r.URL.Path == "/health" {
		h.handleHealthCheck(w, r)
		return
	}

	// Route the request to find the appropriate backend config
	backendConfig, err := h.router.Route(r)
	if err != nil {
		h.handleError(w, r, fmt.Sprintf("routing error: %v", err), http.StatusNotFound)
		return
	}

	// Check if we should strip the path prefix
	shouldStrip, prefix := h.router.ShouldStripPrefix(r)
	if shouldStrip && prefix != "" {
		// Strip the prefix from the path
		r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
	}

	// Get a healthy backend from the load balancer for this backend config
	backend := h.loadBalancer.GetBackendForConfig(backendConfig.Name)
	if backend == nil {
		h.handleError(w, r, "no healthy backends available", http.StatusServiceUnavailable)
		return
	}

	// Create reverse proxy for the selected backend
	proxy := h.createReverseProxy(backend)

	// Add custom headers
	h.addProxyHeaders(r)

	// Wrap the response writer to capture metrics
	wrapper := &responseWrapper{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		size:           0,
	}

	// Forward the request
	proxy.ServeHTTP(wrapper, r)

	// Record metrics
	duration := time.Since(start)
	h.recordMetrics(r, wrapper, backend, duration)

	// Log the request
	h.logRequest(r, wrapper.statusCode, duration, backend.Name)
}

// createReverseProxy creates a reverse proxy for the given backend
func (h *Handler) createReverseProxy(backend *Backend) *httputil.ReverseProxy {
	target, _ := url.Parse(backend.URL) // Error already handled in load balancer

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Customize the director to modify requests
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Modify the request as needed
		req.Host = target.Host
		req.URL.Host = target.Host
		req.URL.Scheme = target.Scheme

		// Add/modify headers
		req.Header.Set("X-Forwarded-Proto", "https")
		req.Header.Set("X-Forwarded-Port", "443")
	}

	// Customize error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logrus.WithFields(logrus.Fields{
			"backend": backend.Name,
			"error":   err.Error(),
			"url":     r.URL.String(),
		}).Error("Backend request failed")

		// Mark backend as unhealthy if it's a connection error
		if strings.Contains(err.Error(), "connection refused") ||
			strings.Contains(err.Error(), "no such host") ||
			strings.Contains(err.Error(), "timeout") {
			backend.SetHealthy(false)
		}

		// Record backend error
		if h.metrics != nil {
			h.metrics.RecordBackendRequest(backend.Name, "error", 0)
		}

		// Return error response
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Backend service unavailable"))
	}

	// Modify response
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Add custom response headers
		resp.Header.Set("X-Proxy-By", "quic-reverse-proxy")
		resp.Header.Set("X-Backend", backend.Name)

		// Record successful backend request
		if h.metrics != nil {
			h.metrics.RecordBackendRequest(backend.Name, "success", 0)
		}

		return nil
	}

	return proxy
}

// addProxyHeaders adds proxy-related headers to the request
func (h *Handler) addProxyHeaders(r *http.Request) {
	// Add X-Forwarded-For header
	if clientIP := h.getClientIP(r); clientIP != "" {
		if existing := r.Header.Get("X-Forwarded-For"); existing != "" {
			r.Header.Set("X-Forwarded-For", existing+", "+clientIP)
		} else {
			r.Header.Set("X-Forwarded-For", clientIP)
		}
	}

	// Add X-Real-IP header
	if clientIP := h.getClientIP(r); clientIP != "" {
		r.Header.Set("X-Real-IP", clientIP)
	}

	// Add X-Forwarded-Proto header
	r.Header.Set("X-Forwarded-Proto", "https")
}

// getClientIP extracts the client IP address from the request
func (h *Handler) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Check X-Real-IP header
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}

	// Fall back to RemoteAddr
	if ip := strings.Split(r.RemoteAddr, ":"); len(ip) > 0 {
		return ip[0]
	}

	return ""
}

// handleError handles proxy errors
func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, message string, statusCode int) {
	logrus.WithFields(logrus.Fields{
		"method":      r.Method,
		"url":         r.URL.String(),
		"remote_addr": r.RemoteAddr,
		"error":       message,
		"status":      statusCode,
	}).Error("Proxy error")

	w.WriteHeader(statusCode)
	w.Write([]byte(message))

	// Record error metrics
	if h.metrics != nil {
		h.metrics.RecordHTTPRequest(
			r.Method,
			"error",
			statusCode,
			0,
			0,
			int64(len(message)),
		)
	}
}

// recordMetrics records request metrics
func (h *Handler) recordMetrics(r *http.Request, w *responseWrapper, backend *Backend, duration time.Duration) {
	if h.metrics == nil {
		return
	}

	// Calculate request size
	requestSize := int64(len(r.Method) + len(r.URL.String()) + len(r.Proto))
	for name, values := range r.Header {
		for _, value := range values {
			requestSize += int64(len(name) + len(value) + 4)
		}
	}
	if r.ContentLength > 0 {
		requestSize += r.ContentLength
	}

	// Record HTTP metrics
	h.metrics.RecordHTTPRequest(
		r.Method,
		backend.Name,
		w.statusCode,
		duration,
		requestSize,
		w.size,
	)
}

// logRequest logs the completed request
func (h *Handler) logRequest(r *http.Request, statusCode int, duration time.Duration, backendName string) {
	logrus.WithFields(logrus.Fields{
		"method":      r.Method,
		"path":        r.URL.Path,
		"status":      statusCode,
		"duration":    duration.String(),
		"backend":     backendName,
		"remote_addr": r.RemoteAddr,
		"user_agent":  r.UserAgent(),
		"referer":     r.Referer(),
	}).Info("Request completed")
}

// responseWrapper wraps http.ResponseWriter to capture response details
type responseWrapper struct {
	http.ResponseWriter
	statusCode int
	size       int64
}

func (w *responseWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWrapper) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	w.size += int64(n)
	return n, err
}

// handleHealthCheck handles the /health endpoint
func (h *Handler) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	// Get backend status from load balancer
	backends := h.loadBalancer.GetBackendStatus()
	
	// Count healthy backends
	healthyCount := 0
	totalCount := len(backends)
	
	for _, backendInfo := range backends {
		if backendMap, ok := backendInfo.(map[string]interface{}); ok {
			if healthy, ok := backendMap["healthy"].(bool); ok && healthy {
				healthyCount++
			}
		}
	}
	
	// Determine overall health status
	status := "healthy"
	statusCode := http.StatusOK
	
	if healthyCount == 0 {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	} else if healthyCount < totalCount {
		status = "degraded"
	}
	
	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	// Simple JSON response
	fmt.Fprintf(w, `{"status":"%s","healthy_backends":%d,"total_backends":%d,"timestamp":"%s"}`,
		status, healthyCount, totalCount, time.Now().Format(time.RFC3339))
	
	logrus.WithFields(logrus.Fields{
		"status":         status,
		"healthy_count":  healthyCount,
		"total_count":    totalCount,
	}).Debug("Health check completed")
}
