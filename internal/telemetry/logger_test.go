package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name      string
		debug     bool
		wantLevel zapcore.Level
	}{
		{
			name:      "debug logger",
			debug:     true,
			wantLevel: zapcore.DebugLevel,
		},
		{
			name:      "production logger",
			debug:     false,
			wantLevel: zapcore.InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.debug)
			require.NoError(t, err)
			assert.NotNil(t, logger)
			assert.NotNil(t, logger.Logger)

			// Replace global logger for test
			oldLogger := zap.L()
			defer zap.ReplaceGlobals(oldLogger)
			zap.ReplaceGlobals(logger.Logger)

			// Verify it was set
			assert.Equal(t, logger.Logger, zap.L())
		})
	}
}

func TestLoggerWithContext(t *testing.T) {
	logger, err := NewLogger(true)
	require.NoError(t, err)

	// Create a real tracer with a noop provider to get recording spans
	tracerProvider := trace.NewNoopTracerProvider()
	tracer := tracerProvider.Tracer("test")

	// For noop provider, spans are not recording, so WithContext returns the same logger
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// Get logger with context
	ctxLogger := logger.WithContext(ctx)
	assert.NotNil(t, ctxLogger)

	// For non-recording spans, should return the same logger
	assert.Equal(t, logger.Logger, ctxLogger)
}

func TestLoggerWithSpanContext(t *testing.T) {
	logger, err := NewLogger(true)
	require.NoError(t, err)

	// Create a mock span context
	ctx := context.Background()

	// Test with no span in context - should return the same logger
	ctxLogger := logger.WithContext(ctx)
	assert.Equal(t, logger.Logger, ctxLogger)

	// With noop tracer, spans are not recording
	tracerProvider := trace.NewNoopTracerProvider()
	tracer := tracerProvider.Tracer("test")
	ctx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	// Test with non-recording span - should still return the same logger
	ctxLogger = logger.WithContext(ctx)
	assert.Equal(t, logger.Logger, ctxLogger)
}

func TestLoggerMethods(t *testing.T) {
	// Create a noop tracer for testing
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	logger, err := NewLogger(true)
	require.NoError(t, err)

	// Test Info level
	assert.NotPanics(t, func() {
		logger.Info(ctx, "test message", zap.String("key", "value"))
	})

	// Test Error level
	assert.NotPanics(t, func() {
		logger.Error(ctx, "error message", zap.Error(assert.AnError))
	})

	// Test Debug level
	assert.NotPanics(t, func() {
		logger.Debug(ctx, "debug message", zap.Int("count", 42))
	})

	// Test Warn level
	assert.NotPanics(t, func() {
		logger.Warn(ctx, "warning message", zap.Float64("ratio", 0.5))
	})
}

func TestSetDefault(t *testing.T) {
	logger, err := NewLogger(true)
	require.NoError(t, err)

	// Store the original global logger
	originalLogger := zap.L()
	defer zap.ReplaceGlobals(originalLogger)

	// Set our logger as default
	logger.SetDefault()

	// Verify it was set
	assert.Equal(t, logger.Logger, zap.L())
}
