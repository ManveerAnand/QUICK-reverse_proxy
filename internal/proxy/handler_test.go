package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/os-dev/quic-reverse-proxy/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// newTestHandler builds the real router + load balancer + handler in front of
// the given backend URLs, all in one "api" pool.
func newTestHandler(t *testing.T, algorithm string, targets ...string) *Handler {
	t.Helper()
	cfg := &config.Config{
		Backends: []config.BackendConfig{{Name: "api", Targets: targets, LoadBalancer: algorithm}},
		Routing: config.RoutingConfig{
			DefaultBackend: "api",
			Rules:          []config.RouteRule{{Path: "/api/*", Backend: "api", Priority: 10}},
		},
	}
	router, err := NewRouter(cfg)
	if err != nil {
		t.Fatal(err)
	}
	lb, err := NewLoadBalancer(cfg.Backends, nil)
	if err != nil {
		t.Fatal(err)
	}
	return NewHandler(router, lb, nil)
}

func TestProxyForwardsAndTraces(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))
	otel.SetTextMapPropagator(propagation.TraceContext{})

	var gotTraceparent, gotXFF string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceparent = r.Header.Get("Traceparent")
		gotXFF = r.Header.Get("X-Forwarded-For")
		io.WriteString(w, "hello from backend")
	}))
	defer backend.Close()

	h := newTestHandler(t, "round_robin", backend.URL)
	req := httptest.NewRequest("GET", "/api/users", nil)
	req.RemoteAddr = "192.0.2.10:5000"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || rec.Body.String() != "hello from backend" {
		t.Fatalf("got %d %q, want 200 from backend", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Proxy-By") != "quic-reverse-proxy" {
		t.Errorf("missing X-Proxy-By response header")
	}
	if gotXFF != "192.0.2.10" {
		t.Errorf("X-Forwarded-For = %q, want client IP 192.0.2.10", gotXFF)
	}
	if gotTraceparent == "" {
		t.Errorf("backend did not receive a traceparent header")
	}

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}
	attrs := map[string]string{}
	for _, kv := range spans[0].Attributes() {
		attrs[string(kv.Key)] = kv.Value.Emit()
	}
	if attrs["http.status_code"] != "200" || attrs["proxy.backend"] == "" {
		t.Errorf("span attributes missing status or backend: %v", attrs)
	}
}

func TestProxyReturns502AndEjectsDeadBackend(t *testing.T) {
	dead := httptest.NewServer(http.NotFoundHandler())
	deadURL := dead.URL
	dead.Close() // connections to it are now refused

	h := newTestHandler(t, "round_robin", deadURL)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/x", nil))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("first request: got %d, want 502", rec.Code)
	}

	// Passive health check marked it down, so the next request gets 503
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/x", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("second request: got %d, want 503", rec.Code)
	}
}
