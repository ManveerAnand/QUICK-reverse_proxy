package proxy

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
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
)

// writeTestCert writes a throwaway self-signed cert for localhost and returns
// the cert and key paths (certs/ is gitignored, so tests can't rely on it).
func writeTestCert(t *testing.T) (string, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	certFile, keyFile := filepath.Join(dir, "server.crt"), filepath.Join(dir, "server.key")
	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		t.Fatal(err)
	}
	return certFile, keyFile
}

func TestAdvertiseHTTP3SetsAltSvc(t *testing.T) {
	// Grab a free UDP port so the expected Alt-Svc value is known
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := pc.LocalAddr().(*net.UDPAddr).Port
	pc.Close()

	certFile, keyFile := writeTestCert(t)
	quicServer, err := quic.NewServer(config.ServerConfig{
		Address:  "127.0.0.1:" + strconv.Itoa(port),
		CertFile: certFile,
		KeyFile:  keyFile,
	}, nil)
	if err != nil {
		t.Fatalf("quic.NewServer: %v", err)
	}
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	go quicServer.Start(ok)
	defer quicServer.Shutdown(context.Background())

	handler := advertiseHTTP3(quicServer, ok)
	want := `h3=":` + strconv.Itoa(port) + `"`

	// The header appears once the QUIC listener is serving
	deadline := time.Now().Add(2 * time.Second)
	for {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest("GET", "https://localhost/", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if got := rec.Header().Get("Alt-Svc"); strings.Contains(got, want) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("Alt-Svc = %q, want it to contain %s", rec.Header().Get("Alt-Svc"), want)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
