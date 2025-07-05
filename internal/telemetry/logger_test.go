package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name      string
		debug     bool
		expectErr bool
	}{
		{
			name:      "debug mode",
			debug:     true,
			expectErr: false,
		},
		{
			name:      "production mode",
			debug:     false,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.debug)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, logger)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, logger)
				assert.NotNil(t, logger.Logger)
				assert.NotNil(t, logger.logr)

				// Test that the logger works
				logger.Info(context.Background(), "test message", "key", "value")
				logger.Debug(context.Background(), "debug message", "debug", true)
			}
		})
	}
}

func TestNewLoggerWithOptions(t *testing.T) {
	tests := []struct {
		name      string
		opts      LoggerOptions
		expectErr bool
	}{
		{
			name: "simple backend",
			opts: LoggerOptions{
				Backend:   BackendSimple,
				Level:     "info",
				Format:    "json",
				Component: "test",
			},
			expectErr: false,
		},
		{
			name: "telemetry backend",
			opts: LoggerOptions{
				Backend:   BackendTelemetry,
				Level:     "debug",
				Format:    "text",
				AddSource: true,
			},
			expectErr: false,
		},
		{
			name: "with component name",
			opts: LoggerOptions{
				Backend:   BackendSimple,
				Level:     "info",
				Format:    "json",
				Component: "my-component",
			},
			expectErr: false,
		},
		{
			name: "text format",
			opts: LoggerOptions{
				Backend: BackendSimple,
				Level:   "info",
				Format:  "text",
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLoggerWithOptions(tt.opts)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, logger)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, logger)
				assert.NotNil(t, logger.Logger)
				assert.NotNil(t, logger.logr)

				// Test basic functionality
				ctx := context.Background()
				logger.Info(ctx, "test info message")
				logger.Debug(ctx, "test debug message")
				logger.Warn(ctx, "test warn message")
				logger.Error(ctx, "test error message", "error", "test error")

				// Test logger methods
				assert.NotNil(t, logger.GetLogr())
				assert.NotNil(t, logger.GetLogrWithContext(ctx))
				assert.NotNil(t, logger.WithError(nil))
				assert.NoError(t, logger.Sync())

				// Test SetDefault
				logger.SetDefault()
			}
		})
	}
}

func TestLoggerBackendAgnostic(t *testing.T) {
	// This test verifies that the logger creation is truly backend-agnostic
	// by testing with different backend configurations

	// Test 1: Create logger with simple backend
	logger1, err := NewLoggerWithOptions(LoggerOptions{
		Backend: BackendSimple,
		Level:   "info",
		Format:  "json",
	})
	require.NoError(t, err)
	require.NotNil(t, logger1)

	// Test 2: Create logger with telemetry backend
	logger2, err := NewLoggerWithOptions(LoggerOptions{
		Backend: BackendTelemetry,
		Level:   "info",
		Format:  "json",
	})
	require.NoError(t, err)
	require.NotNil(t, logger2)

	// Test 3: Register a custom backend and use it
	mockBackend := &MockLoggerBackend{}
	RegisterLoggerBackend("mock", mockBackend)

	logger3, err := NewLoggerWithOptions(LoggerOptions{
		Backend: "mock",
		Level:   "info",
		Format:  "json",
	})
	require.NoError(t, err)
	require.NotNil(t, logger3)

	// Verify the mock backend was called
	assert.Equal(t, 1, mockBackend.CallCount)
}
