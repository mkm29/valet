package telemetry

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/mkm29/valet/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Telemetry holds the telemetry providers and instruments
type Telemetry struct {
	config         *config.TelemetryConfig
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	tracer         oteltrace.Tracer
	meter          metric.Meter
	logger         *Logger
}

// TelemetryOptions configures a Telemetry instance
type TelemetryOptions struct {
	Config *config.TelemetryConfig
	// Add more options as needed in the future
	// For example: custom resource attributes, custom exporters, etc.
}

// NewTelemetry creates a new Telemetry instance with options
func NewTelemetry(ctx context.Context, opts TelemetryOptions) (*Telemetry, error) {
	cfg := opts.Config
	if cfg == nil {
		cfg = config.NewTelemetryConfig()
	}

	if !cfg.Enabled {
		return &Telemetry{config: cfg}, nil
	}

	// Create resource
	res, err := newResource(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Initialize trace provider
	traceProvider, err := initTracerProvider(ctx, cfg, res)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize trace provider: %w", err)
	}

	// Initialize meter provider
	meterProvider, err := initMeterProvider(ctx, cfg, res)
	if err != nil {
		// If meter fails, cleanup trace provider
		if traceProvider != nil {
			_ = traceProvider.Shutdown(context.Background())
		}
		return nil, fmt.Errorf("failed to initialize meter provider: %w", err)
	}

	// Set global providers
	otel.SetTracerProvider(traceProvider)
	otel.SetMeterProvider(meterProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Create tracer and meter
	tracer := otel.Tracer("valet")
	meter := otel.Meter("valet")

	// Create structured logger
	logger, err := NewLogger(cfg.SampleRate > 0) // Use sample rate as debug indicator
	if err != nil {
		// Cleanup providers on error
		_ = meterProvider.Shutdown(context.Background())
		_ = traceProvider.Shutdown(context.Background())
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	return &Telemetry{
		config:         cfg,
		tracerProvider: traceProvider,
		meterProvider:  meterProvider,
		tracer:         tracer,
		meter:          meter,
		logger:         logger,
	}, nil
}

// Initialize initializes the telemetry providers (backward compatibility)
func Initialize(ctx context.Context, cfg *config.TelemetryConfig) (*Telemetry, error) {
	return NewTelemetry(ctx, TelemetryOptions{
		Config: cfg,
	})
}

// NewTelemetryWithConfig creates a new Telemetry instance with just config (convenience function)
func NewTelemetryWithConfig(ctx context.Context, cfg *config.TelemetryConfig) (*Telemetry, error) {
	return NewTelemetry(ctx, TelemetryOptions{
		Config: cfg,
	})
}

// Shutdown shuts down the telemetry providers
func (t *Telemetry) Shutdown(ctx context.Context) error {
	if t == nil || !t.config.Enabled {
		return nil
	}

	var errs []error

	if t.tracerProvider != nil {
		if err := t.tracerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown tracer provider: %w", err))
		}
	}

	if t.meterProvider != nil {
		if err := t.meterProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown meter provider: %w", err))
		}
	}

	// Sync logger
	if t.logger != nil {
		if err := t.logger.Sync(); err != nil {
			errs = append(errs, fmt.Errorf("failed to sync logger: %w", err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// Tracer returns the tracer
func (t *Telemetry) Tracer() oteltrace.Tracer {
	if t == nil || t.tracer == nil {
		return otel.Tracer("valet")
	}
	return t.tracer
}

// Meter returns the meter
func (t *Telemetry) Meter() metric.Meter {
	if t == nil || t.meter == nil {
		return otel.Meter("valet")
	}
	return t.meter
}

// IsEnabled returns true if telemetry is enabled
func (t *Telemetry) IsEnabled() bool {
	return t != nil && t.config != nil && t.config.Enabled
}

// Logger returns the structured logger
func (t *Telemetry) Logger() *Logger {
	if t == nil || t.logger == nil {
		// Return a default logger if telemetry is not initialized
		logger, _ := NewLogger(false)
		return logger
	}
	return t.logger
}

// newResource creates a new resource with service information
func newResource(cfg *config.TelemetryConfig) (*resource.Resource, error) {
	hostname, _ := os.Hostname()

	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			attribute.String("host.name", hostname),
		),
	)
}

// initTracerProvider initializes the tracer provider
func initTracerProvider(ctx context.Context, cfg *config.TelemetryConfig, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	var exporter sdktrace.SpanExporter
	var err error

	switch cfg.ExporterType {
	case "otlp":
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
		}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		if len(cfg.Headers) > 0 {
			opts = append(opts, otlptracegrpc.WithHeaders(cfg.Headers))
		}
		exporter, err = otlptracegrpc.New(ctx, opts...)
	case "stdout":
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	default:
		// No exporter (noop)
		return sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
			sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SampleRate)),
		), nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SampleRate)),
	)

	return tp, nil
}

// initMeterProvider initializes the meter provider
func initMeterProvider(ctx context.Context, cfg *config.TelemetryConfig, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	var exporter sdkmetric.Exporter
	var err error

	switch cfg.ExporterType {
	case "otlp":
		opts := []otlpmetricgrpc.Option{
			otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint),
		}
		if cfg.Insecure {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}
		if len(cfg.Headers) > 0 {
			opts = append(opts, otlpmetricgrpc.WithHeaders(cfg.Headers))
		}
		exporter, err = otlpmetricgrpc.New(ctx, opts...)
	case "stdout":
		exporter, err = stdoutmetric.New()
	default:
		// No exporter (noop)
		return sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
		), nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter,
			sdkmetric.WithInterval(30*time.Second),
		)),
		sdkmetric.WithResource(res),
	)

	return mp, nil
}

// StartSpan starts a new span
func (t *Telemetry) StartSpan(ctx context.Context, name string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	if t == nil || !t.IsEnabled() {
		return ctx, oteltrace.SpanFromContext(ctx)
	}
	return t.tracer.Start(ctx, name, opts...)
}

// RecordError records an error on the span
func RecordError(ctx context.Context, err error, opts ...oteltrace.EventOption) {
	if err == nil {
		return
	}
	span := oteltrace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.RecordError(err, opts...)
	}
}

// SetStatus sets the status of the span
func SetStatus(ctx context.Context, code codes.Code, description string) {
	span := oteltrace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetStatus(code, description)
	}
}

// AddAttributes adds attributes to the span
func AddAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := oteltrace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}
