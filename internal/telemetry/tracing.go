package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Tracer is the OpenTelemetry tracer used for tracing requests.
var Tracer = otel.Tracer("quic-reverse-proxy")

// StartSpan starts a new span for tracing.
func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	ctx, span := Tracer.Start(ctx, name)
	return ctx, span
}

// EndSpan ends the given span and records any errors.
func EndSpan(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
	}
	span.End()
}

// TraceHTTPRequest traces an HTTP request.
func TraceHTTPRequest(req *http.Request) {
	_, span := StartSpan(req.Context(), "HTTP "+req.Method)
	defer EndSpan(span, nil)

	span.SetAttributes(
		attribute.String("http.url", req.URL.String()),
		attribute.String("http.method", req.Method),
		attribute.String("http.remote", req.RemoteAddr),
	)

	// Simulate processing time
	time.Sleep(100 * time.Millisecond)

	// Here you would add your request handling logic
}

// TraceError traces an error with a specific message.
func TraceError(ctx context.Context, message string, err error) {
	_, span := StartSpan(ctx, "Error: "+message)
	defer EndSpan(span, err)
}

// Example usage of tracing in a handler
func ExampleHandler(w http.ResponseWriter, r *http.Request) {
	TraceHTTPRequest(r)

	// Simulate an error
	err := fmt.Errorf("example error")
	if err != nil {
		TraceError(r.Context(), "Handling request", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Hello, World!"))
}