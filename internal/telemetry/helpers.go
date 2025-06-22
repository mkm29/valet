package telemetry

import (
	"context"
	"time"

	"github.com/mkm29/valet/internal/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// CommandMetrics holds metrics for command execution
type CommandMetrics struct {
	ExecutionCounter  metric.Int64Counter
	DurationHistogram metric.Float64Histogram
	ErrorCounter      metric.Int64Counter
}

// NewCommandMetrics creates metrics for command execution
func (t *Telemetry) NewCommandMetrics() (*CommandMetrics, error) {
	if t == nil || !t.IsEnabled() {
		return &CommandMetrics{}, nil
	}

	meter := t.Meter()

	executionCounter, err := meter.Int64Counter(
		"valet.command.executions",
		metric.WithDescription("Total number of command executions"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	durationHistogram, err := meter.Float64Histogram(
		"valet.command.duration",
		metric.WithDescription("Command execution duration"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	errorCounter, err := meter.Int64Counter(
		"valet.command.errors",
		metric.WithDescription("Total number of command errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	return &CommandMetrics{
		ExecutionCounter:  executionCounter,
		DurationHistogram: durationHistogram,
		ErrorCounter:      errorCounter,
	}, nil
}

// RecordCommandExecution records command execution metrics
func (m *CommandMetrics) RecordCommandExecution(ctx context.Context, command string, duration time.Duration, err error) {
	if m == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("command", command),
		attribute.Bool("error", err != nil),
	}

	if m.ExecutionCounter != nil {
		m.ExecutionCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	}

	if m.DurationHistogram != nil {
		m.DurationHistogram.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}

	if err != nil && m.ErrorCounter != nil {
		errorAttrs := append(attrs, attribute.String("error.type", errorType(err)))
		m.ErrorCounter.Add(ctx, 1, metric.WithAttributes(errorAttrs...))
	}
}

// errorType returns a simplified error type for metrics
func errorType(err error) string {
	if err == nil {
		return ""
	}
	// You can add more specific error type detection here
	return "generic"
}

// WithCommandSpan wraps a command execution with a span and metrics
func WithCommandSpan(ctx context.Context, t *Telemetry, command string, fn func(context.Context) error) error {
	start := time.Now()

	// Start span
	ctx, span := t.StartSpan(ctx, "command."+command,
		trace.WithAttributes(
			attribute.String("command", command),
		),
	)
	defer span.End()

	// Execute the function
	err := fn(ctx)

	// Record duration
	duration := time.Since(start)

	// Set span status and attributes
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	span.SetAttributes(
		attribute.Float64("duration.seconds", duration.Seconds()),
	)

	return err
}

// FileOperationMetrics holds metrics for file operations
type FileOperationMetrics struct {
	ReadCounter   metric.Int64Counter
	WriteCounter  metric.Int64Counter
	SizeHistogram metric.Int64Histogram
}

// NewFileOperationMetrics creates metrics for file operations
func (t *Telemetry) NewFileOperationMetrics() (*FileOperationMetrics, error) {
	if t == nil || !t.IsEnabled() {
		return &FileOperationMetrics{}, nil
	}

	meter := t.Meter()

	readCounter, err := meter.Int64Counter(
		"valet.file.reads",
		metric.WithDescription("Total number of file read operations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	writeCounter, err := meter.Int64Counter(
		"valet.file.writes",
		metric.WithDescription("Total number of file write operations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	sizeHistogram, err := meter.Int64Histogram(
		"valet.file.size",
		metric.WithDescription("File size distribution"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	return &FileOperationMetrics{
		ReadCounter:   readCounter,
		WriteCounter:  writeCounter,
		SizeHistogram: sizeHistogram,
	}, nil
}

// RecordFileRead records a file read operation
func (m *FileOperationMetrics) RecordFileRead(ctx context.Context, path string, size int64, err error) {
	if m == nil || m.ReadCounter == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("path", utils.SanitizePath(path)),
		attribute.Bool("error", err != nil),
	}

	m.ReadCounter.Add(ctx, 1, metric.WithAttributes(attrs...))

	if err == nil && m.SizeHistogram != nil {
		m.SizeHistogram.Record(ctx, size, metric.WithAttributes(attrs...))
	}
}

// RecordFileWrite records a file write operation
func (m *FileOperationMetrics) RecordFileWrite(ctx context.Context, path string, size int64, err error) {
	if m == nil || m.WriteCounter == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("path", utils.SanitizePath(path)),
		attribute.Bool("error", err != nil),
	}

	m.WriteCounter.Add(ctx, 1, metric.WithAttributes(attrs...))

	if err == nil && m.SizeHistogram != nil {
		m.SizeHistogram.Record(ctx, size, metric.WithAttributes(attrs...))
	}
}

// SchemaGenerationMetrics holds metrics for schema generation
type SchemaGenerationMetrics struct {
	GenerationCounter metric.Int64Counter
	FieldCounter      metric.Int64Histogram
	DurationHistogram metric.Float64Histogram
}

// NewSchemaGenerationMetrics creates metrics for schema generation
func (t *Telemetry) NewSchemaGenerationMetrics() (*SchemaGenerationMetrics, error) {
	if t == nil || !t.IsEnabled() {
		return &SchemaGenerationMetrics{}, nil
	}

	meter := t.Meter()

	generationCounter, err := meter.Int64Counter(
		"valet.schema.generations",
		metric.WithDescription("Total number of schema generations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	fieldCounter, err := meter.Int64Histogram(
		"valet.schema.fields",
		metric.WithDescription("Number of fields in generated schemas"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	durationHistogram, err := meter.Float64Histogram(
		"valet.schema.generation_duration",
		metric.WithDescription("Schema generation duration"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	return &SchemaGenerationMetrics{
		GenerationCounter: generationCounter,
		FieldCounter:      fieldCounter,
		DurationHistogram: durationHistogram,
	}, nil
}

// RecordSchemaGeneration records schema generation metrics
func (m *SchemaGenerationMetrics) RecordSchemaGeneration(ctx context.Context, fields int64, duration time.Duration, err error) {
	if m == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.Bool("error", err != nil),
	}

	if m.GenerationCounter != nil {
		m.GenerationCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	}

	if err == nil {
		if m.FieldCounter != nil {
			m.FieldCounter.Record(ctx, fields, metric.WithAttributes(attrs...))
		}
		if m.DurationHistogram != nil {
			m.DurationHistogram.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
		}
	}
}
