package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// Tracer is the OpenTelemetry tracer used for tracing requests. It reads the
// global provider on every call, so it starts exporting once tracing is set up
// in NewManager and is a no-op when tracing is disabled.
var Tracer = otel.Tracer("quic-reverse-proxy")

// StartSpan starts a new span as a child of any span already in ctx.
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return Tracer.Start(ctx, name, opts...)
}
