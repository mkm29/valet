package telemetry

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/mkm29/valet/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// openTelemetryProvider implements the Provider interface using OpenTelemetry
type openTelemetryProvider struct {
	tracerProvider TracerProvider
	meterProvider  MeterProvider
	logger         *Logger
	config         *config.TelemetryConfig
}

// newOpenTelemetryProvider creates a new OpenTelemetry provider
func newOpenTelemetryProvider(ctx context.Context, opts ProviderOptions) (Provider, error) {
	cfg, ok := opts.Config.(*config.TelemetryConfig)
	if !ok {
		return nil, fmt.Errorf("invalid config type for OpenTelemetry provider")
	}

	// Create resource
	res, err := newResource(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Initialize tracer provider
	tracerProvider, err := newOtelTracerProvider(ctx, cfg, res)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracer provider: %w", err)
	}

	// Initialize meter provider
	meterProvider, err := newOtelMeterProvider(ctx, cfg, res)
	if err != nil {
		// Cleanup tracer provider on error
		_ = tracerProvider.Shutdown(ctx)
		return nil, fmt.Errorf("failed to initialize meter provider: %w", err)
	}

	// Set global providers with proper type assertion
	if tp, ok := tracerProvider.(*otelTracerProvider); ok {
		otel.SetTracerProvider(tp.provider)
	} else {
		// Log warning if type assertion fails
		if opts.Logger != nil {
			opts.Logger.Warn(ctx, "unexpected tracer provider type", "type", fmt.Sprintf("%T", tracerProvider))
		}
	}
	if mp, ok := meterProvider.(*otelMeterProvider); ok {
		otel.SetMeterProvider(mp.provider)
	} else {
		// Log warning if type assertion fails
		if opts.Logger != nil {
			opts.Logger.Warn(ctx, "unexpected meter provider type", "type", fmt.Sprintf("%T", meterProvider))
		}
	}
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Create logger
	logger := opts.Logger
	if logger == nil {
		logger, err = NewLogger(cfg.SampleRate >= 1.0) // More explicit debug detection
		if err != nil {
			// Best-effort cleanup with timeout
			cleanupCtx, cancel := context.WithTimeout(context.Background(), DefaultBatchTimeout)
			defer cancel()
			if mpErr := meterProvider.Shutdown(cleanupCtx); mpErr != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to shutdown meter provider: %v\n", mpErr)
			}
			if tpErr := tracerProvider.Shutdown(cleanupCtx); tpErr != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to shutdown tracer provider: %v\n", tpErr)
			}
			return nil, fmt.Errorf("failed to create logger: %w", err)
		}
	}

	return &openTelemetryProvider{
		tracerProvider: tracerProvider,
		meterProvider:  meterProvider,
		logger:         logger,
		config:         cfg,
	}, nil
}

// Tracer returns a tracer for creating spans
func (p *openTelemetryProvider) Tracer(name string) trace.Tracer {
	return p.tracerProvider.Tracer(name)
}

// Meter returns a meter for creating metrics
func (p *openTelemetryProvider) Meter(name string) metric.Meter {
	return p.meterProvider.Meter(name)
}

// Logger returns the telemetry-aware logger
func (p *openTelemetryProvider) Logger() *Logger {
	return p.logger
}

// Shutdown gracefully shuts down the provider
func (p *openTelemetryProvider) Shutdown(ctx context.Context) error {
	var errs []error

	if err := p.tracerProvider.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Errorf("failed to shutdown tracer provider: %w", err))
	}

	if err := p.meterProvider.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Errorf("failed to shutdown meter provider: %w", err))
	}

	if p.logger != nil {
		if err := p.logger.Sync(); err != nil {
			errs = append(errs, fmt.Errorf("failed to sync logger: %w", err))
		}
	}

	return errors.Join(errs...)
}

// ForceFlush forces pending telemetry data to be exported
func (p *openTelemetryProvider) ForceFlush(ctx context.Context) error {
	var errs []error

	if err := p.tracerProvider.ForceFlush(ctx); err != nil {
		errs = append(errs, fmt.Errorf("failed to flush tracer provider: %w", err))
	}

	if err := p.meterProvider.ForceFlush(ctx); err != nil {
		errs = append(errs, fmt.Errorf("failed to flush meter provider: %w", err))
	}

	return errors.Join(errs...)
}

// otelTracerProvider wraps the OpenTelemetry tracer provider
type otelTracerProvider struct {
	provider *sdktrace.TracerProvider
}

// newOtelTracerProvider creates a new OpenTelemetry tracer provider
func newOtelTracerProvider(ctx context.Context, cfg *config.TelemetryConfig, res *resource.Resource) (TracerProvider, error) {
	// This reuses the existing initTracerProvider logic
	provider, err := initTracerProvider(ctx, cfg, res)
	if err != nil {
		return nil, err
	}

	return &otelTracerProvider{provider: provider}, nil
}

func (p *otelTracerProvider) Tracer(name string) trace.Tracer {
	return p.provider.Tracer(name)
}

func (p *otelTracerProvider) Shutdown(ctx context.Context) error {
	return p.provider.Shutdown(ctx)
}

func (p *otelTracerProvider) ForceFlush(ctx context.Context) error {
	return p.provider.ForceFlush(ctx)
}

// otelMeterProvider wraps the OpenTelemetry meter provider
type otelMeterProvider struct {
	provider *sdkmetric.MeterProvider
}

// newOtelMeterProvider creates a new OpenTelemetry meter provider
func newOtelMeterProvider(ctx context.Context, cfg *config.TelemetryConfig, res *resource.Resource) (MeterProvider, error) {
	// This reuses the existing initMeterProvider logic
	provider, err := initMeterProvider(ctx, cfg, res)
	if err != nil {
		return nil, err
	}

	return &otelMeterProvider{provider: provider}, nil
}

func (p *otelMeterProvider) Meter(name string) metric.Meter {
	return p.provider.Meter(name)
}

func (p *otelMeterProvider) Shutdown(ctx context.Context) error {
	return p.provider.Shutdown(ctx)
}

func (p *otelMeterProvider) ForceFlush(ctx context.Context) error {
	return p.provider.ForceFlush(ctx)
}

// initTracerProvider and initMeterProvider are defined in telemetry.go
