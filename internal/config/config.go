package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	// Read configuration file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults and validate
	if err := setDefaults(&cfg); err != nil {
		return nil, fmt.Errorf("failed to set defaults: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &cfg, nil
}

// setDefaults applies default values to configuration
func setDefaults(cfg *Config) error {
	// Server defaults
	if cfg.Server.Address == "" {
		cfg.Server.Address = ":443"
	}
	if cfg.Server.FallbackAddress == "" {
		cfg.Server.FallbackAddress = ":80"
	}

	// QUIC defaults
	if cfg.Server.QUIC.MaxStreams == 0 {
		cfg.Server.QUIC.MaxStreams = 1000
	}
	if cfg.Server.QUIC.IdleTimeout == 0 {
		cfg.Server.QUIC.IdleTimeout = 30 * time.Second
	}
	if cfg.Server.QUIC.KeepAlive == 0 {
		cfg.Server.QUIC.KeepAlive = 15 * time.Second
	}
	if cfg.Server.QUIC.MaxTokenAge == 0 {
		cfg.Server.QUIC.MaxTokenAge = 24 * time.Hour
	}
	if cfg.Server.QUIC.CongestionAlgorithm == "" {
		cfg.Server.QUIC.CongestionAlgorithm = "cubic"
	}

	// Backend defaults
	for i := range cfg.Backends {
		backend := &cfg.Backends[i]
		if backend.LoadBalancer == "" {
			backend.LoadBalancer = "round_robin"
		}
		if backend.Weight == 0 {
			backend.Weight = 1
		}
		if backend.Timeout == 0 {
			backend.Timeout = 10 * time.Second
		}
		if backend.RetryCount == 0 {
			backend.RetryCount = 3
		}

		// Health check defaults
		if !backend.HealthCheck.Enabled {
			continue
		}
		if backend.HealthCheck.Path == "" {
			backend.HealthCheck.Path = "/health"
		}
		if backend.HealthCheck.Interval == 0 {
			backend.HealthCheck.Interval = 30 * time.Second
		}
		if backend.HealthCheck.Timeout == 0 {
			backend.HealthCheck.Timeout = 5 * time.Second
		}
		if backend.HealthCheck.HealthyThreshold == 0 {
			backend.HealthCheck.HealthyThreshold = 2
		}
		if backend.HealthCheck.UnhealthyThreshold == 0 {
			backend.HealthCheck.UnhealthyThreshold = 3
		}
	}

	// Telemetry defaults
	if cfg.Telemetry.Metrics.Port == 0 {
		cfg.Telemetry.Metrics.Port = 9090
	}
	if cfg.Telemetry.Metrics.Path == "" {
		cfg.Telemetry.Metrics.Path = "/metrics"
	}
	if cfg.Telemetry.Metrics.CollectionInterval == 0 {
		cfg.Telemetry.Metrics.CollectionInterval = 15 * time.Second
	}

	if cfg.Telemetry.Tracing.ServiceName == "" {
		cfg.Telemetry.Tracing.ServiceName = "quic-reverse-proxy"
	}
	if cfg.Telemetry.Tracing.SampleRate == 0 {
		cfg.Telemetry.Tracing.SampleRate = 0.1 // 10% sampling by default
	}

	if cfg.Telemetry.Logging.Level == "" {
		cfg.Telemetry.Logging.Level = "info"
	}
	if cfg.Telemetry.Logging.Format == "" {
		cfg.Telemetry.Logging.Format = "json"
	}

	return nil
}

// validate checks the configuration for correctness
func validate(cfg *Config) error {
	// Validate server configuration
	if cfg.Server.CertFile == "" {
		return fmt.Errorf("server.cert_file is required")
	}
	if cfg.Server.KeyFile == "" {
		return fmt.Errorf("server.key_file is required")
	}

	// Check if certificate files exist
	if _, err := os.Stat(cfg.Server.CertFile); os.IsNotExist(err) {
		return fmt.Errorf("certificate file does not exist: %s", cfg.Server.CertFile)
	}
	if _, err := os.Stat(cfg.Server.KeyFile); os.IsNotExist(err) {
		return fmt.Errorf("private key file does not exist: %s", cfg.Server.KeyFile)
	}

	// Validate QUIC configuration
	if cfg.Server.QUIC.MaxStreams <= 0 {
		return fmt.Errorf("quic.max_streams must be positive")
	}
	if cfg.Server.QUIC.IdleTimeout <= 0 {
		return fmt.Errorf("quic.idle_timeout must be positive")
	}
	if cfg.Server.QUIC.KeepAlive <= 0 {
		return fmt.Errorf("quic.keep_alive must be positive")
	}

	validAlgorithms := map[string]bool{
		"cubic":   true,
		"bbr":     true,
		"newreno": true,
	}
	if !validAlgorithms[cfg.Server.QUIC.CongestionAlgorithm] {
		return fmt.Errorf("invalid congestion algorithm: %s", cfg.Server.QUIC.CongestionAlgorithm)
	}

	// Validate backends
	if len(cfg.Backends) == 0 {
		return fmt.Errorf("at least one backend must be configured")
	}

	for i, backend := range cfg.Backends {
		if backend.Name == "" {
			return fmt.Errorf("backend[%d].name is required", i)
		}
		if len(backend.Targets) == 0 {
			return fmt.Errorf("backend[%d].targets cannot be empty", i)
		}

		validLBMethods := map[string]bool{
			"round_robin":       true,
			"least_connections": true,
			"weighted":          true,
		}
		if !validLBMethods[backend.LoadBalancer] {
			return fmt.Errorf("invalid load balancer method: %s", backend.LoadBalancer)
		}

		if backend.Weight <= 0 {
			return fmt.Errorf("backend[%d].weight must be positive", i)
		}
		if backend.Timeout <= 0 {
			return fmt.Errorf("backend[%d].timeout must be positive", i)
		}
		if backend.RetryCount < 0 {
			return fmt.Errorf("backend[%d].retry_count cannot be negative", i)
		}
	}

	// Validate telemetry configuration
	if cfg.Telemetry.Metrics.Port <= 0 || cfg.Telemetry.Metrics.Port > 65535 {
		return fmt.Errorf("telemetry.metrics.port must be between 1 and 65535")
	}

	if cfg.Telemetry.Tracing.SampleRate < 0 || cfg.Telemetry.Tracing.SampleRate > 1 {
		return fmt.Errorf("telemetry.tracing.sample_rate must be between 0 and 1")
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[cfg.Telemetry.Logging.Level] {
		return fmt.Errorf("invalid logging level: %s", cfg.Telemetry.Logging.Level)
	}

	validLogFormats := map[string]bool{
		"json": true,
		"text": true,
	}
	if !validLogFormats[cfg.Telemetry.Logging.Format] {
		return fmt.Errorf("invalid logging format: %s", cfg.Telemetry.Logging.Format)
	}

	return nil
}
