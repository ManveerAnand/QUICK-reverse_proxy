package telemetry

import (
	"context"
	"fmt"

	"github.com/os-dev/quic-reverse-proxy/internal/config"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

// Manager coordinates all telemetry components
type Manager struct {
	metrics       *Metrics
	metricsServer *MetricsServer
	tracer        *tracesdk.TracerProvider
	config        config.TelemetryConfig
}

// NewManager creates a new telemetry manager
func NewManager(cfg config.TelemetryConfig) (*Manager, error) {
	manager := &Manager{
		config: cfg,
	}

	// Initialize metrics
	if cfg.Metrics.Enabled {
		manager.metrics = NewMetrics()
		manager.metricsServer = NewMetricsServer(cfg.Metrics.Port, cfg.Metrics.Path)

		// Start metrics server in a goroutine
		go func() {
			if err := manager.metricsServer.Start(); err != nil {
				logrus.WithError(err).Error("Failed to start metrics server")
			}
		}()
	}

	// Initialize tracing
	if cfg.Tracing.Enabled {
		tracer, err := initTracing(cfg.Tracing)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize tracing: %w", err)
		}
		manager.tracer = tracer
	}

	return manager, nil
}

// GetMetrics returns the metrics instance
func (m *Manager) GetMetrics() *Metrics {
	return m.metrics
}

// Shutdown gracefully shuts down all telemetry components
func (m *Manager) Shutdown(ctx context.Context) error {
	var lastErr error

	// Shutdown metrics server
	if m.metricsServer != nil {
		if err := m.metricsServer.Shutdown(ctx); err != nil {
			logrus.WithError(err).Error("Failed to shutdown metrics server")
			lastErr = err
		}
	}

	// Shutdown tracer
	if m.tracer != nil {
		if err := m.tracer.Shutdown(ctx); err != nil {
			logrus.WithError(err).Error("Failed to shutdown tracer")
			lastErr = err
		}
	}

	return lastErr
}

// initTracing initializes OpenTelemetry tracing
func initTracing(cfg config.TracingConfig) (*tracesdk.TracerProvider, error) {
	// Create Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(cfg.Endpoint)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// Create tracer provider with resource
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(cfg.ServiceName),
		)),
		tracesdk.WithSampler(tracesdk.TraceIDRatioBased(cfg.SampleRate)),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	logrus.WithFields(logrus.Fields{
		"endpoint":     cfg.Endpoint,
		"service_name": cfg.ServiceName,
		"sample_rate":  cfg.SampleRate,
	}).Info("Tracing initialized")

	return tp, nil
}
