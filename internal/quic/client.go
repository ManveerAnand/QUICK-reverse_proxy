package quic

import (
	"crypto/tls"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

// NewRoundTripper creates an http3.RoundTripper that can be used by an httputil.ReverseProxy
// to route traffic to HTTP/3 enabled backend services.
func NewRoundTripper(tlsSkipVerify bool) *http3.RoundTripper {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: tlsSkipVerify,
		NextProtos:         []string{"h3"}, // Explicitly request HTTP/3 ALPN
	}

	quicConfig := &quic.Config{}

	return &http3.RoundTripper{
		TLSClientConfig: tlsConfig,
		QuicConfig:      quicConfig,
	}
}
