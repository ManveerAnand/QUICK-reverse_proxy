package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// Metrics holds all Prometheus metrics for the QUIC reverse proxy
type Metrics struct {
	// Connection metrics
	QUICConnections        *prometheus.GaugeVec
	QUICConnectionDuration *prometheus.HistogramVec
	QUICHandshakeDuration  *prometheus.HistogramVec
	QUIC0RTTAttempts       *prometheus.CounterVec

	// Transport metrics
	QUICPackets          *prometheus.CounterVec
	QUICPacketLossRate   *prometheus.GaugeVec
	QUICRTT              *prometheus.GaugeVec
	QUICCongestionWindow *prometheus.GaugeVec

	// Request metrics
	HTTPRequests        *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestSize     *prometheus.HistogramVec
	HTTPResponseSize    *prometheus.HistogramVec

	// Backend metrics
	BackendRequests     *prometheus.CounterVec
	BackendResponseTime *prometheus.HistogramVec
	BackendHealthStatus *prometheus.GaugeVec
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics() *Metrics {
	m := &Metrics{
		// Connection metrics
		QUICConnections: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "quic_connections_total",
				Help: "Total number of QUIC connections by state",
			},
			[]string{"state"}, // active, closed, failed
		),

		QUICConnectionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "quic_connection_duration_seconds",
				Help:    "Duration of QUIC connections",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"reason"}, // normal, timeout, error
		),

		QUICHandshakeDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "quic_handshake_duration_seconds",
				Help:    "Duration of QUIC handshakes",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"cipher_suite", "protocol_version", "zero_rtt"},
		),

		QUIC0RTTAttempts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "quic_zero_rtt_attempts_total",
				Help: "Total number of 0-RTT attempts",
			},
			[]string{"result"}, // success, failed
		),

		// Transport metrics
		QUICPackets: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "quic_packets_total",
				Help: "Total number of QUIC packets by direction and type",
			},
			[]string{"direction", "type"}, // direction: sent/received, type: initial/handshake/1rtt
		),

		QUICPacketLossRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "quic_packet_loss_rate",
				Help: "Current packet loss rate",
			},
			[]string{"connection_id"},
		),

		QUICRTT: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "quic_rtt_seconds",
				Help: "Current round-trip time in seconds",
			},
			[]string{"connection_id"},
		),

		QUICCongestionWindow: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "quic_congestion_window_bytes",
				Help: "Current congestion window size in bytes",
			},
			[]string{"connection_id", "algorithm"},
		),

		// Request metrics
		HTTPRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "status_code", "backend"},
		),

		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "Duration of HTTP requests",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"method", "backend"},
		),

		HTTPRequestSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_size_bytes",
				Help:    "Size of HTTP requests in bytes",
				Buckets: prometheus.ExponentialBuckets(1024, 2, 10), // 1KB to 512KB
			},
			[]string{"method", "backend"},
		),

		HTTPResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_response_size_bytes",
				Help:    "Size of HTTP responses in bytes",
				Buckets: prometheus.ExponentialBuckets(1024, 2, 15), // 1KB to 16MB
			},
			[]string{"status_code", "backend"},
		),

		// Backend metrics
		BackendRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "backend_requests_total",
				Help: "Total number of backend requests",
			},
			[]string{"backend", "status"},
		),

		BackendResponseTime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "backend_response_time_seconds",
				Help:    "Backend response time",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"backend"},
		),

		BackendHealthStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "backend_health_status",
				Help: "Backend health status (1=healthy, 0=unhealthy)",
			},
			[]string{"backend"},
		),
	}

	// Register all metrics with Prometheus
	prometheus.MustRegister(
		m.QUICConnections,
		m.QUICConnectionDuration,
		m.QUICHandshakeDuration,
		m.QUIC0RTTAttempts,
		m.QUICPackets,
		m.QUICPacketLossRate,
		m.QUICRTT,
		m.QUICCongestionWindow,
		m.HTTPRequests,
		m.HTTPRequestDuration,
		m.HTTPRequestSize,
		m.HTTPResponseSize,
		m.BackendRequests,
		m.BackendResponseTime,
		m.BackendHealthStatus,
	)

	return m
}

// RecordConnection records connection-related metrics
func (m *Metrics) RecordConnection(state string) {
	m.QUICConnections.WithLabelValues(state).Inc()
}

// RecordConnectionClosed records when a connection is closed
func (m *Metrics) RecordConnectionClosed(duration time.Duration, reason string) {
	m.QUICConnections.WithLabelValues("active").Dec()
	m.QUICConnections.WithLabelValues("closed").Inc()
	m.QUICConnectionDuration.WithLabelValues(reason).Observe(duration.Seconds())
}

// RecordHandshake records handshake metrics
func (m *Metrics) RecordHandshake(duration time.Duration, cipherSuite, protocolVersion string, is0RTT bool) {
	zeroRTT := "false"
	if is0RTT {
		zeroRTT = "true"
	}
	m.QUICHandshakeDuration.WithLabelValues(cipherSuite, protocolVersion, zeroRTT).Observe(duration.Seconds())
}

// Record0RTTAttempt records 0-RTT attempt results
func (m *Metrics) Record0RTTAttempt(success bool) {
	result := "failed"
	if success {
		result = "success"
	}
	m.QUIC0RTTAttempts.WithLabelValues(result).Inc()
}

// RecordPacket records packet metrics
func (m *Metrics) RecordPacket(direction, packetType string) {
	m.QUICPackets.WithLabelValues(direction, packetType).Inc()
}

// UpdateTransportMetrics updates transport-level metrics
func (m *Metrics) UpdateTransportMetrics(connectionID, algorithm string, rtt time.Duration, packetLoss float64, congestionWindow int) {
	m.QUICRTT.WithLabelValues(connectionID).Set(rtt.Seconds())
	m.QUICPacketLossRate.WithLabelValues(connectionID).Set(packetLoss)
	m.QUICCongestionWindow.WithLabelValues(connectionID, algorithm).Set(float64(congestionWindow))
}

// RecordHTTPRequest records HTTP request metrics
func (m *Metrics) RecordHTTPRequest(method, backend string, statusCode int, duration time.Duration, requestSize, responseSize int64) {
	statusStr := strconv.Itoa(statusCode)

	m.HTTPRequests.WithLabelValues(method, statusStr, backend).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, backend).Observe(duration.Seconds())
	m.HTTPRequestSize.WithLabelValues(method, backend).Observe(float64(requestSize))
	m.HTTPResponseSize.WithLabelValues(statusStr, backend).Observe(float64(responseSize))
}

// RecordBackendRequest records backend-specific metrics
func (m *Metrics) RecordBackendRequest(backend, status string, responseTime time.Duration) {
	m.BackendRequests.WithLabelValues(backend, status).Inc()
	m.BackendResponseTime.WithLabelValues(backend).Observe(responseTime.Seconds())
}

// UpdateBackendHealth updates backend health status
func (m *Metrics) UpdateBackendHealth(backend string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	m.BackendHealthStatus.WithLabelValues(backend).Set(value)
}

// MetricsServer provides HTTP endpoint for Prometheus metrics
type MetricsServer struct {
	server *http.Server
	port   int
	path   string
}

// NewMetricsServer creates a new metrics server
func NewMetricsServer(port int, path string) *MetricsServer {
	mux := http.NewServeMux()
	mux.Handle(path, promhttp.Handler())

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	return &MetricsServer{
		server: server,
		port:   port,
		path:   path,
	}
}

// Start starts the metrics server
func (ms *MetricsServer) Start() error {
	logrus.WithFields(logrus.Fields{
		"port": ms.port,
		"path": ms.path,
	}).Info("Starting metrics server")

	if err := ms.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("metrics server failed: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the metrics server
func (ms *MetricsServer) Shutdown(ctx context.Context) error {
	logrus.Info("Shutting down metrics server")
	return ms.server.Shutdown(ctx)
}
