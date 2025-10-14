package config

import (
	"time"
)

// Config represents the complete application configuration
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Backends  []BackendConfig `yaml:"backends"`
	Telemetry TelemetryConfig `yaml:"telemetry"`
}

// ServerConfig contains QUIC server configuration
type ServerConfig struct {
	Address         string     `yaml:"address"`
	CertFile        string     `yaml:"cert_file"`
	KeyFile         string     `yaml:"key_file"`
	QUIC            QUICConfig `yaml:"quic"`
	FallbackAddress string     `yaml:"fallback_address,omitempty"`
}

// QUICConfig contains QUIC-specific settings
type QUICConfig struct {
	MaxStreams          int           `yaml:"max_streams"`
	IdleTimeout         time.Duration `yaml:"idle_timeout"`
	KeepAlive           time.Duration `yaml:"keep_alive"`
	Enable0RTT          bool          `yaml:"enable_0rtt"`
	MaxTokenAge         time.Duration `yaml:"max_token_age,omitempty"`
	CongestionAlgorithm string        `yaml:"congestion_algorithm,omitempty"` // "cubic", "bbr", "newreno"
}

// BackendConfig represents a backend service configuration
type BackendConfig struct {
	Name         string            `yaml:"name"`
	Targets      []string          `yaml:"targets"`
	HealthCheck  HealthCheckConfig `yaml:"health_check"`
	LoadBalancer string            `yaml:"load_balancer"` // "round_robin", "least_connections", "weighted"
	Weight       int               `yaml:"weight,omitempty"`
	Timeout      time.Duration     `yaml:"timeout,omitempty"`
	RetryCount   int               `yaml:"retry_count,omitempty"`
}

// HealthCheckConfig contains health check settings
type HealthCheckConfig struct {
	Enabled            bool          `yaml:"enabled"`
	Path               string        `yaml:"path"`
	Interval           time.Duration `yaml:"interval"`
	Timeout            time.Duration `yaml:"timeout"`
	HealthyThreshold   int           `yaml:"healthy_threshold,omitempty"`
	UnhealthyThreshold int           `yaml:"unhealthy_threshold,omitempty"`
}

// TelemetryConfig contains telemetry configuration
type TelemetryConfig struct {
	Metrics MetricsConfig `yaml:"metrics"`
	Tracing TracingConfig `yaml:"tracing"`
	Logging LoggingConfig `yaml:"logging"`
}

// MetricsConfig contains Prometheus metrics configuration
type MetricsConfig struct {
	Enabled            bool          `yaml:"enabled"`
	Port               int           `yaml:"port"`
	Path               string        `yaml:"path"`
	CollectionInterval time.Duration `yaml:"collection_interval,omitempty"`
}

// TracingConfig contains distributed tracing configuration
type TracingConfig struct {
	Enabled     bool    `yaml:"enabled"`
	Endpoint    string  `yaml:"endpoint"`
	ServiceName string  `yaml:"service_name,omitempty"`
	SampleRate  float64 `yaml:"sample_rate,omitempty"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`            // "debug", "info", "warn", "error"
	Format string `yaml:"format"`           // "json", "text"
	Output string `yaml:"output,omitempty"` // file path or "stdout"
}
