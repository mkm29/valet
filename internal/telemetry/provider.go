package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Provider defines the interface for telemetry providers
// This allows for different implementations (OpenTelemetry, Jaeger, etc.)
type Provider interface {
	// Tracer returns a tracer for creating spans
	Tracer(name string) trace.Tracer

	// Meter returns a meter for creating metrics
	Meter(name string) metric.Meter

	// Logger returns a telemetry-aware logger
	Logger() *Logger

	// Shutdown gracefully shuts down the provider
	Shutdown(ctx context.Context) error

	// ForceFlush forces pending telemetry data to be exported
	ForceFlush(ctx context.Context) error
}

// TracerProvider defines the interface for trace providers
type TracerProvider interface {
	// Tracer returns a tracer with the given name
	Tracer(name string) trace.Tracer

	// Shutdown shuts down the tracer provider
	Shutdown(ctx context.Context) error

	// ForceFlush forces pending spans to be exported
	ForceFlush(ctx context.Context) error
}

// MeterProvider defines the interface for metrics providers
type MeterProvider interface {
	// Meter returns a meter with the given name
	Meter(name string) metric.Meter

	// Shutdown shuts down the meter provider
	Shutdown(ctx context.Context) error

	// ForceFlush forces pending metrics to be exported
	ForceFlush(ctx context.Context) error
}

// ProviderType represents the type of telemetry provider
type ProviderType string

const (
	ProviderTypeOpenTelemetry ProviderType = "opentelemetry"
	ProviderTypeNoop          ProviderType = "noop"
	// Future providers can be added here
	// ProviderTypeJaeger ProviderType = "jaeger"
	// ProviderTypeZipkin ProviderType = "zipkin"
)

// ProviderOptions configures the telemetry provider
type ProviderOptions struct {
	Type     ProviderType
	Config   interface{} // Provider-specific configuration
	Logger   *Logger
	Exporter string // "stdout", "otlp", "none"
}

// NewProvider creates a new telemetry provider based on options
func NewProvider(ctx context.Context, opts ProviderOptions) (Provider, error) {
	switch opts.Type {
	case ProviderTypeOpenTelemetry:
		return newOpenTelemetryProvider(ctx, opts)
	case ProviderTypeNoop:
		return newNoopProvider(), nil
	default:
		// Default to no-op provider
		return newNoopProvider(), nil
	}
}
