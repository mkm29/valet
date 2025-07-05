package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/metric"
	noop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
)

// noopProvider implements Provider interface with no-op operations
type noopProvider struct {
	tracer trace.Tracer
	meter  metric.Meter
	logger *Logger
}

// newNoopProvider creates a new no-op provider
func newNoopProvider() Provider {
	return &noopProvider{
		tracer: nooptrace.NewTracerProvider().Tracer("noop"),
		meter:  noop.NewMeterProvider().Meter("noop"),
		logger: nil, // Can be set later if needed
	}
}

// Tracer returns a no-op tracer
func (p *noopProvider) Tracer(name string) trace.Tracer {
	return p.tracer
}

// Meter returns a no-op meter
func (p *noopProvider) Meter(name string) metric.Meter {
	return p.meter
}

// Logger returns the logger (may be nil for no-op)
func (p *noopProvider) Logger() *Logger {
	return p.logger
}

// Shutdown is a no-op
func (p *noopProvider) Shutdown(ctx context.Context) error {
	return nil
}

// ForceFlush is a no-op
func (p *noopProvider) ForceFlush(ctx context.Context) error {
	return nil
}

// noopMetricsCollector implements MetricsCollector with no-op operations
type noopMetricsCollector struct{}

// newNoopMetricsCollector creates a new no-op metrics collector
func newNoopMetricsCollector() MetricsCollector {
	return &noopMetricsCollector{}
}

// RecordCommandExecution is a no-op
func (c *noopMetricsCollector) RecordCommandExecution(ctx context.Context, command string, duration time.Duration, err error) {
	// No-op
}

// RecordFileOperation is a no-op
func (c *noopMetricsCollector) RecordFileOperation(operation string, size int64, err error) {
	// No-op
}

// RecordSchemaGeneration is a no-op
func (c *noopMetricsCollector) RecordSchemaGeneration(fields int, duration time.Duration, err error) {
	// No-op
}

// RecordHelmCacheStats is a no-op
func (c *noopMetricsCollector) RecordHelmCacheStats(stats CacheStatsProvider) {
	// No-op
}

// Start is a no-op
func (c *noopMetricsCollector) Start(ctx context.Context) error {
	return nil
}

// Stop is a no-op
func (c *noopMetricsCollector) Stop(ctx context.Context) error {
	return nil
}
