package proxy

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/os-dev/quic-reverse-proxy/internal/config"
	"github.com/os-dev/quic-reverse-proxy/internal/quic"
	"github.com/os-dev/quic-reverse-proxy/internal/telemetry"
	"github.com/sirupsen/logrus"
)

// Server represents the main reverse proxy server
type Server struct {
	config       *config.Config
	quicServer   *quic.Server
	httpServer   *http.Server
	router       *Router
	handler      *Handler
	telemetry    *telemetry.Manager
	loadBalancer *LoadBalancer
}

// NewServer creates a new reverse proxy server
func NewServer(cfg *config.Config, telemetryManager *telemetry.Manager) (*Server, error) {
	// Create router
	router, err := NewRouter(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create router: %w", err)
	}

	// Create load balancer
	loadBalancer, err := NewLoadBalancer(cfg.Backends)
	if err != nil {
		return nil, fmt.Errorf("failed to create load balancer: %w", err)
	}

	// Create proxy handler with router
	handler := NewHandler(router, loadBalancer, telemetryManager.GetMetrics())

	// Create QUIC server
	quicServer, err := quic.NewServer(cfg.Server, telemetryManager.GetMetrics())
	if err != nil {
		return nil, fmt.Errorf("failed to create QUIC server: %w", err)
	}

	// Create HTTP fallback server (for testing and compatibility)
	httpServer := &http.Server{
		Addr:         cfg.Server.FallbackAddress,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		config:       cfg,
		quicServer:   quicServer,
		httpServer:   httpServer,
		router:       router,
		handler:      handler,
		telemetry:    telemetryManager,
		loadBalancer: loadBalancer,
	}, nil
}

// Start starts the reverse proxy server
func (s *Server) Start() error {
	// Start health checks
	s.loadBalancer.StartHealthChecks()

	logrus.WithFields(logrus.Fields{
		"address":  s.config.Server.Address,
		"backends": len(s.config.Backends),
	}).Info("Starting reverse proxy server")

	// Start HTTP fallback server in a goroutine
	if s.httpServer != nil {
		go func() {
			logrus.WithField("address", s.httpServer.Addr).Info("Starting HTTP fallback server")
			if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logrus.WithError(err).Error("HTTP fallback server error")
			}
		}()
	}

	// Start the QUIC server with our handler
	return s.quicServer.Start(s.handler)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	logrus.Info("Shutting down reverse proxy server")

	// Stop health checks
	s.loadBalancer.StopHealthChecks()

	// Shutdown HTTP fallback server
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			logrus.WithError(err).Error("Failed to shutdown HTTP fallback server")
		}
	}

	// Shutdown QUIC server
	return s.quicServer.Shutdown(ctx)
}

// GetMetrics returns the server metrics for monitoring
func (s *Server) GetMetrics() *telemetry.Metrics {
	return s.telemetry.GetMetrics()
}

// ReloadConfig reloads the server configuration
func (s *Server) ReloadConfig(newConfig *config.Config) error {
	logrus.Info("Reloading server configuration")

	// Update backend configuration
	if err := s.loadBalancer.UpdateBackends(newConfig.Backends); err != nil {
		return fmt.Errorf("failed to update backends: %w", err)
	}

	// Update configuration
	s.config = newConfig

	logrus.Info("Configuration reloaded successfully")
	return nil
}

// HealthCheck returns the server health status
func (s *Server) HealthCheck() map[string]interface{} {
	status := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"uptime":    time.Since(time.Now()).String(), // Simplified - in real implementation, track actual uptime
		"backends":  s.loadBalancer.GetBackendStatus(),
	}

	return status
}
