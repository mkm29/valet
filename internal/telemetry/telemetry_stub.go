//go:build notelemetry
// +build notelemetry

package telemetry

import (
	"context"

	"github.com/mkm29/valet/internal/config"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	oteltrace "go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

// Telemetry is a no-op implementation when telemetry is disabled
type Telemetry struct {
	config *config.TelemetryConfig
}

// Initialize returns a stub telemetry instance
func Initialize(ctx context.Context, cfg *config.TelemetryConfig) (*Telemetry, error) {
	if cfg == nil {
		cfg = config.NewTelemetryConfig()
		cfg.Enabled = false
	}
	return &Telemetry{config: cfg}, nil
}

// Shutdown is a no-op
func (t *Telemetry) Shutdown(ctx context.Context) error {
	return nil
}

// Tracer returns a no-op tracer
func (t *Telemetry) Tracer() oteltrace.Tracer {
	return nooptrace.NewTracerProvider().Tracer("valet")
}

// Meter returns a no-op meter
func (t *Telemetry) Meter() metric.Meter {
	return noop.NewMeterProvider().Meter("valet")
}

// IsEnabled always returns false for stub
func (t *Telemetry) IsEnabled() bool {
	return false
}

// Logger returns a no-op logger
func (t *Telemetry) Logger() *Logger {
	return &Logger{logger: zap.NewNop()}
}

// StartSpan returns a no-op span
func (t *Telemetry) StartSpan(ctx context.Context, name string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	return nooptrace.NewTracerProvider().Tracer("valet").Start(ctx, name, opts...)
}

// RecordError is a no-op
func RecordError(ctx context.Context, err error, opts ...oteltrace.EventOption) {}

// SetStatus is a no-op
func SetStatus(ctx context.Context, code codes.Code, description string) {}

// AddAttributes is a no-op
func AddAttributes(ctx context.Context, attrs ...attribute.KeyValue) {}
