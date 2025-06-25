package tests

import (
	"context"
	"log/slog"
	"testing"

	"github.com/mkm29/valet/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/trace"
)

type TelemetryLoggerTestSuite struct {
	ValetTestSuite
}

func (suite *TelemetryLoggerTestSuite) TestNewLogger() {
	tests := []struct {
		name      string
		debug     bool
		wantLevel slog.Level
	}{
		{
			name:      "debug logger",
			debug:     true,
			wantLevel: slog.LevelDebug,
		},
		{
			name:      "production logger",
			debug:     false,
			wantLevel: slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			logger, err := telemetry.NewLogger(tt.debug)
			suite.NoError(err)
			suite.NotNil(logger)
			suite.NotNil(logger.Logger)

			// Replace global logger for test
			oldLogger := slog.Default()
			defer slog.SetDefault(oldLogger)
			slog.SetDefault(logger.Logger)

			// Verify it was set
			suite.Equal(logger.Logger, slog.Default())
		})
	}
}

func (suite *TelemetryLoggerTestSuite) TestLoggerWithContext() {
	logger, err := telemetry.NewLogger(true)
	suite.NoError(err)

	// Create a real tracer with a noop provider to get recording spans
	tracerProvider := trace.NewNoopTracerProvider()
	tracer := tracerProvider.Tracer("test")

	// For noop provider, spans are not recording, so WithContext returns the same logger
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// Get logger with context
	ctxLogger := logger.WithContext(ctx)
	suite.NotNil(ctxLogger)

	// For non-recording spans, should return the same logger
	suite.Equal(logger.Logger, ctxLogger)
}

func (suite *TelemetryLoggerTestSuite) TestLoggerWithSpanContext() {
	logger, err := telemetry.NewLogger(true)
	suite.NoError(err)

	// Create a mock span context
	ctx := context.Background()

	// Test with no span in context - should return the same logger
	ctxLogger := logger.WithContext(ctx)
	suite.Equal(logger.Logger, ctxLogger)

	// With noop tracer, spans are not recording
	tracerProvider := trace.NewNoopTracerProvider()
	tracer := tracerProvider.Tracer("test")
	ctx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	// Test with non-recording span - should still return the same logger
	ctxLogger = logger.WithContext(ctx)
	suite.Equal(logger.Logger, ctxLogger)
}

func (suite *TelemetryLoggerTestSuite) TestLoggerMethods() {
	// Create a noop tracer for testing
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	logger, err := telemetry.NewLogger(true)
	suite.NoError(err)

	// Test Info level
	suite.NotPanics(func() {
		logger.Info(ctx, "test message", "key", "value")
	})

	// Test Error level
	suite.NotPanics(func() {
		logger.Error(ctx, "error message", "error", assert.AnError)
	})

	// Test Debug level
	suite.NotPanics(func() {
		logger.Debug(ctx, "debug message", "count", 42)
	})

	// Test Warn level
	suite.NotPanics(func() {
		logger.Warn(ctx, "warning message", "ratio", 0.5)
	})
}

func (suite *TelemetryLoggerTestSuite) TestSetDefault() {
	logger, err := telemetry.NewLogger(true)
	suite.NoError(err)

	// Store the original global logger
	originalLogger := slog.Default()
	defer slog.SetDefault(originalLogger)

	// Set our logger as default
	logger.SetDefault()

	// Verify it was set
	suite.Equal(logger.Logger, slog.Default())
}

func TestTelemetryLoggerSuite(t *testing.T) {
	suite.Run(t, new(TelemetryLoggerTestSuite))
}
