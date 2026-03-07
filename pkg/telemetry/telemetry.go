// Package telemetry provides OpenTelemetry integration for VC Stack.
//
// It sets up a trace provider with configurable exporters (OTLP HTTP/gRPC,
// stdout for dev) and provides Gin middleware for automatic request tracing.
//
// Usage:
//
//	shutdown, err := telemetry.Init(telemetry.Config{
//	    ServiceName: "vc-management",
//	    Endpoint:    "localhost:4318",         // OTLP HTTP
//	    Enabled:     true,
//	})
//	defer shutdown(ctx)
//
//	// Add Gin middleware
//	router.Use(telemetry.GinMiddleware("vc-management"))
package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/gin-gonic/gin"
)

// Config holds OpenTelemetry configuration.
type Config struct {
	// ServiceName identifies this service in traces (e.g. "vc-management").
	ServiceName string

	// ServiceVersion is the deployment version.
	ServiceVersion string

	// Enabled controls whether tracing is active. If false, a noop provider is used.
	Enabled bool

	// Endpoint is the OTLP collector endpoint (e.g. "localhost:4318" for HTTP).
	// Leave empty to use stdout exporter (useful for development).
	Endpoint string

	// SampleRate controls the fraction of traces sampled (0.0 to 1.0).
	// Default: 1.0 (sample everything) in dev, recommend 0.1 in production.
	SampleRate float64

	// Insecure disables TLS for the OTLP connection.
	Insecure bool
}

// ShutdownFunc is returned by Init and should be called on application shutdown.
type ShutdownFunc func(ctx context.Context) error

// Init initializes the OpenTelemetry trace provider.
// Returns a shutdown function that must be called to flush pending spans.
func Init(cfg Config) (ShutdownFunc, error) {
	if !cfg.Enabled {
		// Noop provider — zero overhead when tracing is disabled.
		return func(_ context.Context) error { return nil }, nil
	}

	if cfg.SampleRate <= 0 {
		cfg.SampleRate = 1.0
	}

	ctx := context.Background()

	// Build resource with service metadata.
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			attribute.String("deployment.environment", "production"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("telemetry resource: %w", err)
	}

	// Choose exporter.
	var exporter sdktrace.SpanExporter
	if cfg.Endpoint != "" {
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(cfg.Endpoint),
		}
		if cfg.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		exp, err := otlptracehttp.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("otlp exporter: %w", err)
		}
		exporter = exp
	} else {
		// Stdout exporter for development.
		exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("stdout exporter: %w", err)
		}
		exporter = exp
	}

	// Build trace provider.
	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRate))
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Register as global provider.
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return func(ctx context.Context) error {
		shutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return tp.Shutdown(shutCtx)
	}, nil
}

// GinMiddleware returns Gin middleware for automatic HTTP request tracing.
// It creates a span for each request with method, path, status code attributes.
func GinMiddleware(serviceName string) gin.HandlerFunc {
	return otelgin.Middleware(serviceName)
}

// Tracer returns a named tracer for creating custom spans.
//
// Usage:
//
//	ctx, span := telemetry.Tracer("compute").Start(ctx, "CreateInstance")
//	defer span.End()
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// SpanFromContext extracts the current span from a context.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// AddEvent adds an event to the current span.
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	trace.SpanFromContext(ctx).AddEvent(name, trace.WithAttributes(attrs...))
}

// SetError marks the current span as errored.
func SetError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
}
