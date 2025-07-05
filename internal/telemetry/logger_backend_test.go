package telemetry_test

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/mkm29/valet/internal/telemetry"
	"github.com/stretchr/testify/assert"
)

func TestNewLoggerFromOptions(t *testing.T) {
	tests := []struct {
		name string
		opts telemetry.LoggerOptions
	}{
		{
			name: "default options",
			opts: telemetry.DefaultLoggerOptions(),
		},
		{
			name: "debug logger with simple backend",
			opts: telemetry.LoggerOptions{
				Backend:   telemetry.BackendSimple,
				Level:     "debug",
				Format:    "text",
				AddSource: true,
			},
		},
		{
			name: "production logger with simple backend",
			opts: telemetry.LoggerOptions{
				Backend:   telemetry.BackendSimple,
				Level:     "info",
				Format:    "json",
				AddSource: false,
			},
		},
		{
			name: "with component",
			opts: telemetry.LoggerOptions{
				Backend:   telemetry.BackendSimple,
				Level:     "info",
				Format:    "json",
				Component: "test-component",
			},
		},
		{
			name: "telemetry backend",
			opts: telemetry.LoggerOptions{
				Backend: telemetry.BackendTelemetry,
				Level:   "info",
				Format:  "json",
			},
		},
		{
			name: "empty backend defaults to simple",
			opts: telemetry.LoggerOptions{
				Level:  "info",
				Format: "json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := telemetry.NewLoggerFromOptions(tt.opts)
			assert.NotNil(t, logger)
			assert.True(t, logger.Enabled())
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("NewDebugLogger", func(t *testing.T) {
		logger := telemetry.NewDebugLogger()
		assert.NotNil(t, logger)
		assert.True(t, logger.Enabled())
	})

	t.Run("NewProductionLogger", func(t *testing.T) {
		logger := telemetry.NewProductionLogger()
		assert.NotNil(t, logger)
		assert.True(t, logger.Enabled())
	})

	t.Run("NewTelemetryLogger", func(t *testing.T) {
		logger := telemetry.NewTelemetryLogger("test")
		assert.NotNil(t, logger)
		assert.True(t, logger.Enabled())
	})
}

func TestLoggerLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error", "invalid"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			opts := telemetry.LoggerOptions{
				Backend: telemetry.BackendSimple,
				Level:   level,
				Format:  "json",
			}
			logger := telemetry.NewLoggerFromOptions(opts)
			assert.NotNil(t, logger)
		})
	}
}

func TestMockLoggerBackend(t *testing.T) {
	// Register the mock backend
	mock := telemetry.RegisterMockLoggerBackend()

	// Create a logger with the mock backend
	opts := telemetry.LoggerOptions{
		Backend:   "mock",
		Level:     "debug",
		Format:    "json",
		Component: "test",
	}

	logger := telemetry.NewLoggerFromOptions(opts)

	// Verify the mock was called
	assert.Equal(t, 1, mock.CallCount)
	assert.Equal(t, opts, mock.LastOptions)
	assert.NotNil(t, logger)

	// Test with custom logger function
	mock.CreateLoggerFunc = func(o telemetry.LoggerOptions) logr.Logger {
		// Return a real logger for testing
		return telemetry.NewDebugLogger()
	}

	logger2 := telemetry.NewLoggerFromOptions(opts)
	assert.Equal(t, 2, mock.CallCount)
	assert.True(t, logger2.Enabled())
}

func TestRegisterLoggerBackend(t *testing.T) {
	// Create a custom mock backend
	customMock := &telemetry.MockLoggerBackend{}

	// Register it with a custom name
	telemetry.RegisterLoggerBackend("custom", customMock)

	// Use the custom backend
	opts := telemetry.LoggerOptions{
		Backend: "custom",
		Level:   "info",
		Format:  "text",
	}

	logger := telemetry.NewLoggerFromOptions(opts)
	assert.NotNil(t, logger)
	assert.Equal(t, 1, customMock.CallCount)
	assert.Equal(t, opts, customMock.LastOptions)
}
