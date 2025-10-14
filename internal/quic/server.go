package quic

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"time"

	"github.com/os-dev/quic-reverse-proxy/internal/config"
	"github.com/os-dev/quic-reverse-proxy/internal/telemetry"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/sirupsen/logrus"
)

// Server represents a QUIC HTTP/3 server
type Server struct {
	config    config.ServerConfig
	metrics   *telemetry.Metrics
	server    *http3.Server
	listener  net.PacketConn
	tlsConfig *tls.Config
}

// NewServer creates a new QUIC server instance
func NewServer(cfg config.ServerConfig, metrics *telemetry.Metrics) (*Server, error) {
	// Load TLS configuration
	tlsConfig, err := loadTLSConfig(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS config: %w", err)
	}

	// Create packet connection
	addr, err := net.ResolveUDPAddr("udp", cfg.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve address: %w", err)
	}

	listener, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on UDP: %w", err)
	}

	// Create QUIC config
	quicConfig := &quic.Config{
		MaxIdleTimeout:        cfg.QUIC.IdleTimeout,
		KeepAlivePeriod:       cfg.QUIC.KeepAlive,
		MaxIncomingStreams:    int64(cfg.QUIC.MaxStreams),
		MaxIncomingUniStreams: int64(cfg.QUIC.MaxStreams / 4),
		Allow0RTT:             cfg.QUIC.Enable0RTT,
		TokenStore:            quic.NewLRUTokenStore(100, 100),
		EnableDatagrams:       true,
	}

	// Create HTTP/3 server
	server := &http3.Server{
		Addr:       cfg.Address,
		TLSConfig:  tlsConfig,
		QuicConfig: quicConfig,
	}

	return &Server{
		config:    cfg,
		metrics:   metrics,
		server:    server,
		listener:  listener,
		tlsConfig: tlsConfig,
	}, nil
}

// Start begins listening for incoming QUIC connections
func (s *Server) Start(handler http.Handler) error {
	logrus.WithField("address", s.config.Address).Info("Starting QUIC server")

	// Set the handler
	s.server.Handler = s.wrapHandler(handler)

	// Set up connection callbacks for telemetry
	s.server.QuicConfig.GetConfigForClient = s.getConfigForClient

	// Start listening
	return s.server.Serve(s.listener)
}

// Shutdown gracefully shuts down the QUIC server
func (s *Server) Shutdown(ctx context.Context) error {
	logrus.Info("Shutting down QUIC server")

	// Close the packet connection
	if err := s.listener.Close(); err != nil {
		return fmt.Errorf("failed to close listener: %w", err)
	}

	return nil
}

// wrapHandler wraps the HTTP handler with telemetry and logging
func (s *Server) wrapHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Extract connection information for metrics
		connectionID := s.extractConnectionID(r)

		// Record connection if new
		if s.metrics != nil && connectionID != "" {
			s.metrics.RecordConnection("active")
		}

		// Create a response writer wrapper to capture status and size
		wrappedWriter := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Call the handler
		handler.ServeHTTP(wrappedWriter, r)

		// Record metrics
		if s.metrics != nil {
			duration := time.Since(start)
			requestSize := s.getRequestSize(r)
			responseSize := wrappedWriter.size

			s.metrics.RecordHTTPRequest(
				r.Method,
				"", // backend will be filled by proxy handler
				wrappedWriter.statusCode,
				duration,
				requestSize,
				responseSize,
			)
		}

		// Log request
		logrus.WithFields(logrus.Fields{
			"method":      r.Method,
			"path":        r.URL.Path,
			"status":      wrappedWriter.statusCode,
			"duration":    time.Since(start).String(),
			"remote_addr": r.RemoteAddr,
			"user_agent":  r.UserAgent(),
			"protocol":    r.Proto,
		}).Info("HTTP request processed")
	})
}

// getConfigForClient returns QUIC configuration for incoming connections
func (s *Server) getConfigForClient(info *quic.ClientHelloInfo) (*quic.Config, error) {
	// Record handshake metrics
	if s.metrics != nil {
		// This is a simplified approach - in a real implementation,
		// you'd capture actual handshake timing and details
		go func() {
			time.Sleep(time.Millisecond) // Simulate handshake time
			s.metrics.RecordHandshake(
				time.Millisecond*10,      // Simulated handshake duration
				"TLS_AES_128_GCM_SHA256", // Would be actual cipher suite
				"QUICv1",                 // Would be actual QUIC version
				false,                    // Would check if 0-RTT was used
			)
		}()
	}

	return s.server.QuicConfig, nil
}

// extractConnectionID extracts connection ID from request context
func (s *Server) extractConnectionID(r *http.Request) string {
	// In a real implementation, you'd extract this from the QUIC connection
	// This is a simplified version
	return r.RemoteAddr
}

// getRequestSize calculates the size of the HTTP request
func (s *Server) getRequestSize(r *http.Request) int64 {
	size := int64(len(r.Method) + len(r.URL.String()) + len(r.Proto))

	// Add headers size
	for name, values := range r.Header {
		for _, value := range values {
			size += int64(len(name) + len(value) + 4) // +4 for ": " and "\r\n"
		}
	}

	// Add content length
	if r.ContentLength > 0 {
		size += r.ContentLength
	}

	return size
}

// loadTLSConfig loads TLS configuration from certificate files
func loadTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load key pair: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h3"},
		MinVersion:   tls.VersionTLS13, // QUIC requires TLS 1.3
	}, nil
}

// generateSelfSignedCert generates a self-signed certificate for testing
func generateSelfSignedCert() (tls.Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"QUIC Reverse Proxy"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1)},
		DNSNames:    []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	return tls.X509KeyPair(certPEM, keyPEM)
}

// responseWriterWrapper wraps http.ResponseWriter to capture response details
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
	size       int64
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriterWrapper) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	w.size += int64(n)
	return n, err
}
