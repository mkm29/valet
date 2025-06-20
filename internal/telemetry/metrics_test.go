package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/mkm29/valet/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandMetrics(t *testing.T) {
	// Initialize telemetry
	cfg := &config.TelemetryConfig{
		Enabled:        true,
		ExporterType:   "none",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}
	ctx := context.Background()
	tel, err := Initialize(ctx, cfg)
	require.NoError(t, err)
	require.NotNil(t, tel)
	defer tel.Shutdown(ctx)

	// Create command metrics
	metrics, err := tel.NewCommandMetrics()
	require.NoError(t, err)
	assert.NotNil(t, metrics)

	// Test recording command execution without error
	assert.NotPanics(t, func() {
		metrics.RecordCommandExecution(ctx, "test-command", 100*time.Millisecond, nil)
	})

	// Test recording command execution with error
	assert.NotPanics(t, func() {
		metrics.RecordCommandExecution(ctx, "test-command", 200*time.Millisecond, assert.AnError)
	})

	// Test with nil metrics
	var nilMetrics *CommandMetrics
	assert.NotPanics(t, func() {
		nilMetrics.RecordCommandExecution(ctx, "test", time.Second, nil)
	})
}

func TestFileOperationMetrics(t *testing.T) {
	// Initialize telemetry
	cfg := &config.TelemetryConfig{
		Enabled:        true,
		ExporterType:   "none",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}
	ctx := context.Background()
	tel, err := Initialize(ctx, cfg)
	require.NoError(t, err)
	require.NotNil(t, tel)
	defer tel.Shutdown(ctx)

	// Create file operation metrics
	metrics, err := tel.NewFileOperationMetrics()
	require.NoError(t, err)
	assert.NotNil(t, metrics)

	// Test recording file read
	assert.NotPanics(t, func() {
		metrics.RecordFileRead(ctx, "/test/file.yaml", 1024, nil)
	})

	// Test recording file read with error
	assert.NotPanics(t, func() {
		metrics.RecordFileRead(ctx, "/test/file.yaml", 0, assert.AnError)
	})

	// Test recording file write
	assert.NotPanics(t, func() {
		metrics.RecordFileWrite(ctx, "/test/output.json", 2048, nil)
	})

	// Test with nil metrics
	var nilMetrics *FileOperationMetrics
	assert.NotPanics(t, func() {
		nilMetrics.RecordFileRead(ctx, "test", 100, nil)
		nilMetrics.RecordFileWrite(ctx, "test", 100, nil)
	})
}

func TestSchemaGenerationMetrics(t *testing.T) {
	// Initialize telemetry
	cfg := &config.TelemetryConfig{
		Enabled:        true,
		ExporterType:   "none",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}
	ctx := context.Background()
	tel, err := Initialize(ctx, cfg)
	require.NoError(t, err)
	require.NotNil(t, tel)
	defer tel.Shutdown(ctx)

	// Create schema generation metrics
	metrics, err := tel.NewSchemaGenerationMetrics()
	require.NoError(t, err)
	assert.NotNil(t, metrics)

	// Test recording schema generation
	assert.NotPanics(t, func() {
		metrics.RecordSchemaGeneration(ctx, 50, 300*time.Millisecond, nil)
	})

	// Test recording schema generation with error
	assert.NotPanics(t, func() {
		metrics.RecordSchemaGeneration(ctx, 0, 100*time.Millisecond, assert.AnError)
	})

	// Test with nil metrics
	var nilMetrics *SchemaGenerationMetrics
	assert.NotPanics(t, func() {
		nilMetrics.RecordSchemaGeneration(ctx, 10, time.Second, nil)
	})
}

func TestWithCommandSpan(t *testing.T) {
	// Initialize telemetry
	cfg := &config.TelemetryConfig{
		Enabled:        true,
		ExporterType:   "none",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}
	ctx := context.Background()
	tel, err := Initialize(ctx, cfg)
	require.NoError(t, err)
	require.NotNil(t, tel)
	defer tel.Shutdown(ctx)

	// Test successful command execution
	err = WithCommandSpan(ctx, tel, "test-command", func(ctx context.Context) error {
		// Simulate some work
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	assert.NoError(t, err)

	// Test command execution with error
	expectedErr := assert.AnError
	err = WithCommandSpan(ctx, tel, "failing-command", func(ctx context.Context) error {
		return expectedErr
	})
	assert.Equal(t, expectedErr, err)
}

func TestErrorType(t *testing.T) {
	assert.Equal(t, "", errorType(nil))
	assert.Equal(t, "generic", errorType(assert.AnError))
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "empty path",
			path:     "",
			expected: "",
		},
		{
			name:     "simple filename",
			path:     "file.txt",
			expected: "file.txt",
		},
		{
			name:     "filename with parent directory",
			path:     "parent/file.txt",
			expected: "parent/file.txt",
		},
		{
			name:     "absolute path",
			path:     "/Users/john/Documents/project/file.txt",
			expected: "project/file.txt",
		},
		{
			name:     "home directory path",
			path:     "~/Documents/project/file.txt",
			expected: "project/file.txt",
		},
		{
			name:     "deeply nested path",
			path:     "/var/log/application/service/2024/01/app.log",
			expected: "01/app.log",
		},
		{
			name:     "root file",
			path:     "/file.txt",
			expected: "file.txt",
		},
		{
			name:     "current directory file",
			path:     "./file.txt",
			expected: "file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizePath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}
