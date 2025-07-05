package telemetry

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/mkm29/valet/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestNewProvider(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		opts         ProviderOptions
		expectError  bool
		expectedType string
	}{
		{
			name: "OpenTelemetry provider with disabled telemetry",
			opts: ProviderOptions{
				Type: ProviderTypeOpenTelemetry,
				Config: &config.TelemetryConfig{
					Enabled: false,
				},
			},
			expectError:  false,
			expectedType: "*telemetry.openTelemetryProvider",
		},
		{
			name: "No-op provider",
			opts: ProviderOptions{
				Type: ProviderTypeNoop,
			},
			expectError:  false,
			expectedType: "*telemetry.noopProvider",
		},
		{
			name: "Default to no-op provider",
			opts: ProviderOptions{
				Type: "unknown",
			},
			expectError:  false,
			expectedType: "*telemetry.noopProvider",
		},
		{
			name: "OpenTelemetry provider with stdout exporter",
			opts: ProviderOptions{
				Type: ProviderTypeOpenTelemetry,
				Config: &config.TelemetryConfig{
					Enabled:      true,
					ServiceName:  "test-service",
					ExporterType: "stdout",
				},
			},
			expectError:  false,
			expectedType: "*telemetry.openTelemetryProvider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(ctx, tt.opts)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)

				// Check the actual type
				actualType := getTypeName(provider)
				assert.Equal(t, tt.expectedType, actualType)

				// Test basic provider functionality
				tracer := provider.Tracer("test")
				assert.NotNil(t, tracer)

				meter := provider.Meter("test")
				assert.NotNil(t, meter)

				// Cleanup
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err = provider.Shutdown(shutdownCtx)
				assert.NoError(t, err)
			}
		})
	}
}

func TestNoopProvider(t *testing.T) {
	provider := newNoopProvider()
	assert.NotNil(t, provider)

	// Test Tracer
	tracer := provider.Tracer("test")
	assert.NotNil(t, tracer)
	// Verify it's a no-op tracer by checking if spans are recorded
	_, span := tracer.Start(context.Background(), "test-span")
	assert.NotNil(t, span)
	assert.False(t, span.IsRecording())
	span.End()

	// Test Meter
	meter := provider.Meter("test")
	assert.NotNil(t, meter)

	// Test Logger
	logger := provider.Logger()
	assert.Nil(t, logger) // No-op provider returns nil logger

	// Test Shutdown
	err := provider.Shutdown(context.Background())
	assert.NoError(t, err)

	// Test ForceFlush
	err = provider.ForceFlush(context.Background())
	assert.NoError(t, err)
}

func TestMockProvider(t *testing.T) {
	mock := NewMockProvider()
	assert.NotNil(t, mock)

	// Test Tracer
	tracer := mock.Tracer("test-tracer")
	assert.NotNil(t, tracer)
	assert.Equal(t, 1, mock.TracerCallCount)
	assert.Contains(t, mock.TracerNames, "test-tracer")

	// Test custom tracer function
	customTracer := trace.NewNoopTracerProvider().Tracer("custom")
	mock.TracerFunc = func(name string) trace.Tracer {
		return customTracer
	}
	tracer2 := mock.Tracer("test-tracer-2")
	assert.Equal(t, customTracer, tracer2)

	// Test Meter
	meter := mock.Meter("test-meter")
	assert.NotNil(t, meter)
	assert.Equal(t, 1, mock.MeterCallCount)
	assert.Contains(t, mock.MeterNames, "test-meter")

	// Test Logger
	logger := mock.Logger()
	assert.Nil(t, logger) // Default returns nil
	assert.Equal(t, 1, mock.LoggerCallCount)

	// Test custom logger function
	testLogger, _ := NewLogger(false)
	mock.LoggerFunc = func() *Logger {
		return testLogger
	}
	logger2 := mock.Logger()
	assert.Equal(t, testLogger, logger2)

	// Test Shutdown
	err := mock.Shutdown(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, mock.ShutdownCallCount)

	// Test shutdown error
	mock.ShutdownFunc = nil // Clear the function to use the error field
	mock.ShutdownError = errors.New("shutdown failed")
	err = mock.Shutdown(context.Background())
	assert.Error(t, err)
	assert.Equal(t, "shutdown failed", err.Error())

	// Test ForceFlush
	err = mock.ForceFlush(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, mock.ForceFlushCallCount)

	// Test Reset
	mock.Reset()
	assert.Equal(t, 0, mock.TracerCallCount)
	assert.Equal(t, 0, mock.MeterCallCount)
	assert.Equal(t, 0, mock.LoggerCallCount)
	assert.Equal(t, 0, mock.ShutdownCallCount)
	assert.Equal(t, 0, mock.ForceFlushCallCount)
	assert.Empty(t, mock.TracerNames)
	assert.Empty(t, mock.MeterNames)
}

func TestMockTracerProvider(t *testing.T) {
	mock := NewMockTracerProvider()
	assert.NotNil(t, mock)

	// Test Tracer
	tracer := mock.Tracer("test")
	assert.NotNil(t, tracer)
	assert.Equal(t, 1, mock.TracerCallCount)
	assert.Contains(t, mock.TracerNames, "test")

	// Test Shutdown
	err := mock.Shutdown(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, mock.ShutdownCallCount)

	// Test ForceFlush
	err = mock.ForceFlush(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, mock.ForceFlushCallCount)
}

func TestMockMeterProvider(t *testing.T) {
	mock := NewMockMeterProvider()
	assert.NotNil(t, mock)

	// Test Meter
	meter := mock.Meter("test")
	assert.NotNil(t, meter)
	assert.Equal(t, 1, mock.MeterCallCount)
	assert.Contains(t, mock.MeterNames, "test")

	// Test Shutdown
	err := mock.Shutdown(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, mock.ShutdownCallCount)

	// Test ForceFlush
	err = mock.ForceFlush(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, mock.ForceFlushCallCount)
}

func TestOpenTelemetryProviderWithMocks(t *testing.T) {
	// This test demonstrates how the interface design allows for easy testing
	// by using mock providers in the OpenTelemetry provider

	ctx := context.Background()

	// Create a disabled config to avoid actual provider initialization
	cfg := &config.TelemetryConfig{
		Enabled: false,
	}

	provider, err := newOpenTelemetryProvider(ctx, ProviderOptions{
		Config: cfg,
	})
	require.NoError(t, err)
	require.NotNil(t, provider)

	// Even with disabled telemetry, we should get valid (no-op) tracers and meters
	tracer := provider.Tracer("test")
	assert.NotNil(t, tracer)

	meter := provider.Meter("test")
	assert.NotNil(t, meter)

	// Logger might be nil for disabled telemetry
	_ = provider.Logger()
	// No assertion on logger as it depends on initialization

	// Shutdown should work even with disabled telemetry
	err = provider.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestProviderInterfaceCompliance(t *testing.T) {
	// This test ensures all provider implementations satisfy the Provider interface
	var _ Provider = (*openTelemetryProvider)(nil)
	var _ Provider = (*noopProvider)(nil)
	var _ Provider = (*MockProvider)(nil)

	// Test TracerProvider interface compliance
	var _ TracerProvider = (*otelTracerProvider)(nil)
	var _ TracerProvider = (*MockTracerProvider)(nil)

	// Test MeterProvider interface compliance
	var _ MeterProvider = (*otelMeterProvider)(nil)
	var _ MeterProvider = (*MockMeterProvider)(nil)
}

func TestProviderWithContext(t *testing.T) {
	// Test providers with context cancellation
	ctx, cancel := context.WithCancel(context.Background())

	provider, err := NewProvider(ctx, ProviderOptions{
		Type: ProviderTypeNoop,
	})
	require.NoError(t, err)
	require.NotNil(t, provider)

	// Cancel the context
	cancel()

	// Operations should still work (no-op provider doesn't use context)
	tracer := provider.Tracer("test")
	assert.NotNil(t, tracer)

	meter := provider.Meter("test")
	assert.NotNil(t, meter)

	// Shutdown with cancelled context should still work
	err = provider.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestProviderFactoryPattern(t *testing.T) {
	// Test that the factory pattern correctly creates providers based on type
	ctx := context.Background()

	testCases := []struct {
		providerType ProviderType
		expectedType string
	}{
		{ProviderTypeOpenTelemetry, "*telemetry.openTelemetryProvider"},
		{ProviderTypeNoop, "*telemetry.noopProvider"},
		{"invalid-type", "*telemetry.noopProvider"}, // Should default to no-op
	}

	for _, tc := range testCases {
		t.Run(string(tc.providerType), func(t *testing.T) {
			provider, err := NewProvider(ctx, ProviderOptions{
				Type: tc.providerType,
				Config: &config.TelemetryConfig{
					Enabled: false, // Disable to avoid actual initialization
				},
			})

			assert.NoError(t, err)
			assert.NotNil(t, provider)
			assert.Equal(t, tc.expectedType, getTypeName(provider))

			// Cleanup
			_ = provider.Shutdown(ctx)
		})
	}
}

// Helper function to get the type name of an interface
func getTypeName(i interface{}) string {
	if i == nil {
		return "nil"
	}
	return typeName(i)
}

func typeName(i interface{}) string {
	t := reflect.TypeOf(i)
	if t.Kind() == reflect.Ptr {
		return t.String()
	}
	return t.Name()
}
