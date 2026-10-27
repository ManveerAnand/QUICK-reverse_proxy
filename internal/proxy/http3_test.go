package proxy

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/os-dev/quic-reverse-proxy/internal/config"
	"github.com/os-dev/quic-reverse-proxy/internal/quic"
	"github.com/quic-go/quic-go/http3"
)

// TestHTTP3EndToEnd sends a real HTTP/3 request over QUIC/UDP to the proxy
// and checks it reaches an HTTP/1.1 backend and comes back.
func TestHTTP3EndToEnd(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "backend saw "+r.URL.Path)
	}))
	defer backend.Close()

	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := pc.LocalAddr().String()
	pc.Close()

	certFile, keyFile := writeTestCert(t)
	quicServer, err := quic.NewServer(config.ServerConfig{
		Address:  addr,
		CertFile: certFile,
		KeyFile:  keyFile,
		QUIC:     config.QUICConfig{MaxStreams: 100, IdleTimeout: 5 * time.Second, KeepAlive: time.Second},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	go quicServer.Start(newTestHandler(t, "round_robin", backend.URL))
	defer quicServer.Shutdown(context.Background())

	client := &http.Client{
		Transport: &http3.RoundTripper{TLSClientConfig: &tls.Config{InsecureSkipVerify: true, NextProtos: []string{"h3"}}},
		Timeout:   3 * time.Second,
	}
	defer client.Transport.(*http3.RoundTripper).Close()

	resp, err := client.Get("https://" + addr + "/api/hello")
	if err != nil {
		t.Fatalf("HTTP/3 request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.ProtoMajor != 3 {
		t.Errorf("client used %s, want HTTP/3", resp.Proto)
	}
	if resp.StatusCode != http.StatusOK || string(body) != "backend saw /api/hello" {
		t.Fatalf("got %d %q", resp.StatusCode, body)
	}
}

// TestDemoConfigLoads makes sure configs/proxy.yaml passes validation
// (cert paths are swapped for a temporary cert, since certs/ is gitignored).
func TestDemoConfigLoads(t *testing.T) {
	raw, err := os.ReadFile("../../configs/proxy.yaml")
	if err != nil {
		t.Fatal(err)
	}
	certFile, keyFile := writeTestCert(t)
	yaml := strings.NewReplacer(
		`"certs/server.crt"`, strconv.Quote(filepath.ToSlash(certFile)),
		`"certs/server.key"`, strconv.Quote(filepath.ToSlash(keyFile)),
	).Replace(string(raw))
	path := filepath.Join(t.TempDir(), "proxy.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("configs/proxy.yaml is invalid: %v", err)
	}
	if _, err := NewRouter(cfg); err != nil {
		t.Fatalf("router: %v", err)
	}
	if _, err := NewLoadBalancer(cfg.Backends, nil); err != nil {
		t.Fatalf("load balancer: %v", err)
	}
}
