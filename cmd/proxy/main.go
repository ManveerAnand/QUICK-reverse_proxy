package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/os-dev/quic-reverse-proxy/internal/config"
	"github.com/os-dev/quic-reverse-proxy/internal/proxy"
	"github.com/os-dev/quic-reverse-proxy/internal/telemetry"
	"github.com/sirupsen/logrus"
)

var (
	configPath = flag.String("config", "configs/proxy.yaml", "Path to configuration file")
	version    = flag.Bool("version", false, "Show version information")
	debug      = flag.Bool("debug", false, "Enable debug logging")
)

const (
	AppName    = "QUIC Reverse Proxy"
	AppVersion = "1.0.0"
)

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("%s v%s\n", AppName, AppVersion)
		fmt.Printf("Built with Go %s\n", "1.21+")
		os.Exit(0)
	}

	// Configure logging
	setupLogging(*debug)

	logrus.WithFields(logrus.Fields{
		"app":     AppName,
		"version": AppVersion,
	}).Info("Starting QUIC Reverse Proxy")

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load configuration")
	}

	logrus.WithField("config_path", *configPath).Info("Configuration loaded successfully")

	// Initialize telemetry
	telemetryManager, err := telemetry.NewManager(cfg.Telemetry)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize telemetry")
	}
	defer telemetryManager.Shutdown(context.Background())

	// Initialize proxy server
	proxyServer, err := proxy.NewServer(cfg, telemetryManager)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create proxy server")
	}

	// Start the server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logrus.WithField("address", cfg.Server.Address).Info("Starting QUIC proxy server")
		serverErrors <- proxyServer.Start()
	}()

	// Wait for interrupt signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		logrus.WithError(err).Error("Server error")

	case sig := <-shutdown:
		logrus.WithField("signal", sig.String()).Info("Shutdown signal received")

		// Create a context with timeout for graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Gracefully shutdown the server
		if err := proxyServer.Shutdown(ctx); err != nil {
			logrus.WithError(err).Error("Server shutdown failed")
		}
	}

	logrus.Info("QUIC Reverse Proxy stopped")
}

func setupLogging(debug bool) {
	// Set log format
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	// Set log level
	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	// Add hooks for structured logging
	logrus.AddHook(&contextHook{})
}

// contextHook adds context information to log entries
type contextHook struct{}

func (h *contextHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *contextHook) Fire(entry *logrus.Entry) error {
	// Add process information
	entry.Data["pid"] = os.Getpid()

	// Add timestamp if not present
	if _, ok := entry.Data["timestamp"]; !ok {
		entry.Data["timestamp"] = time.Now().Format(time.RFC3339)
	}

	return nil
}
