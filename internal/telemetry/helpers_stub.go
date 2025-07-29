//go:build notelemetry
// +build notelemetry

package telemetry

import (
	"context"
	"time"
)

// CommandMetrics is a no-op implementation
type CommandMetrics struct{}

// NewCommandMetrics returns a no-op command metrics
func (t *Telemetry) NewCommandMetrics() (*CommandMetrics, error) {
	return &CommandMetrics{}, nil
}

// RecordCommandExecution is a no-op
func (m *CommandMetrics) RecordCommandExecution(ctx context.Context, command string, duration time.Duration, err error) {
	// No-op implementation
}

// FileOperationMetrics is a no-op implementation
type FileOperationMetrics struct{}

// NewFileOperationMetrics returns a no-op file operation metrics
func (t *Telemetry) NewFileOperationMetrics() (*FileOperationMetrics, error) {
	return &FileOperationMetrics{}, nil
}

// RecordFileRead is a no-op
func (m *FileOperationMetrics) RecordFileRead(ctx context.Context, filename string, size int64, err error) {
	// No-op implementation
}

// RecordFileWrite is a no-op
func (m *FileOperationMetrics) RecordFileWrite(ctx context.Context, filename string, size int64, err error) {
	// No-op implementation
}

// SchemaGenerationMetrics is a no-op implementation
type SchemaGenerationMetrics struct{}

// NewSchemaGenerationMetrics returns a no-op schema generation metrics
func (t *Telemetry) NewSchemaGenerationMetrics() (*SchemaGenerationMetrics, error) {
	return &SchemaGenerationMetrics{}, nil
}

// RecordSchemaGeneration is a no-op
func (m *SchemaGenerationMetrics) RecordSchemaGeneration(ctx context.Context, fieldCount int64, duration time.Duration, err error) {
	// No-op implementation
}

// WithCommandSpan is a no-op wrapper
func WithCommandSpan(ctx context.Context, t *Telemetry, command string, fn func(context.Context) error) error {
	return fn(ctx)
}