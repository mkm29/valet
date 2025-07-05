package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/mkm29/valet/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TestTelemetryWithInterfaces tests the existing Telemetry struct works with new interfaces
func TestTelemetryWithInterfaces(t *testing.T) {
	ctx := context.Background()

	// Test with disabled telemetry
	t.Run("Disabled telemetry", func(t *testing.T) {
		cfg := &config.TelemetryConfig{
			Enabled: false,
		}

		tel, err := NewTelemetry(ctx, TelemetryOptions{Config: cfg})
		assert.NoError(t, err)
		assert.NotNil(t, tel)

		// Should return no-op implementations
		assert.NotNil(t, tel.Tracer())
		assert.NotNil(t, tel.Meter())

		// Shutdown should work
		err = tel.Shutdown(ctx)
		assert.NoError(t, err)
	})

	// Test with stdout exporter
	t.Run("Stdout exporter", func(t *testing.T) {
		cfg := &config.TelemetryConfig{
			Enabled:      true,
			ServiceName:  "test-service",
			ExporterType: "stdout",
			SampleRate:   1.0,
		}

		tel, err := NewTelemetry(ctx, TelemetryOptions{Config: cfg})
		assert.NoError(t, err)
		assert.NotNil(t, tel)

		// Should have valid tracer and meter
		tracer := tel.Tracer()
		assert.NotNil(t, tracer)

		meter := tel.Meter()
		assert.NotNil(t, meter)

		// Create a span
		_, span := tracer.Start(ctx, "test-span")
		assert.NotNil(t, span)
		span.End()

		// Shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err = tel.Shutdown(shutdownCtx)
		assert.NoError(t, err)
	})
}

// TestMetricsServerWithInterfaces tests the MetricsServer with interface-based collectors
func TestMetricsServerWithInterfaces(t *testing.T) {
	// Test creating a Prometheus metrics server
	cfg := &config.MetricsConfig{
		Enabled: true,
		Port:    0, // Random port
		Path:    "/metrics",
	}

	server, err := NewPrometheusMetricsServer(cfg, NewDebugLogger())
	require.NoError(t, err)
	require.NotNil(t, server)

	ctx := context.Background()

	// Record some metrics
	server.RecordCommandExecution(ctx, "test", time.Second, nil)
	server.RecordFileOperation("read", 1024, nil)
	server.RecordSchemaGeneration(10, time.Millisecond*100, nil)

	// Test the handler
	handler := server.Handler()
	assert.NotNil(t, handler)

	// Test the endpoint
	endpoint := server.Endpoint()
	assert.Equal(t, "/metrics", endpoint)

	// Stop the server
	err = server.Stop(ctx)
	assert.NoError(t, err)
}

// TestProviderIntegration tests how providers integrate with existing code
func TestProviderIntegration(t *testing.T) {
	ctx := context.Background()

	// Create a provider using the factory
	provider, err := NewProvider(ctx, ProviderOptions{
		Type: ProviderTypeOpenTelemetry,
		Config: &config.TelemetryConfig{
			Enabled:      true,
			ServiceName:  "integration-test",
			ExporterType: "stdout",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, provider)

	// Use the provider
	tracer := provider.Tracer("test-component")
	assert.NotNil(t, tracer)

	meter := provider.Meter("test-component")
	assert.NotNil(t, meter)

	_ = provider.Logger()
	// Logger might be nil or not depending on initialization

	// Create a span
	ctx, span := tracer.Start(ctx, "integration-test-span")
	assert.NotNil(t, span)

	// Add some attributes
	span.SetAttributes(
		attribute.String("test.type", "integration"),
		attribute.Int("test.id", 123),
	)

	span.End()

	// Shutdown
	err = provider.Shutdown(ctx)
	assert.NoError(t, err)
}

// TestMetricsCollectorIntegration tests metrics collector integration
func TestMetricsCollectorIntegration(t *testing.T) {
	ctx := context.Background()

	// Create a metrics collector
	collector, err := NewMetricsCollector(MetricsCollectorOptions{
		Type: MetricsCollectorTypePrometheus,
		Config: &config.MetricsConfig{
			Enabled: true,
			Port:    0,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, collector)

	// Record various metrics
	collector.RecordCommandExecution(ctx, "generate", time.Second*2, nil)
	collector.RecordCommandExecution(ctx, "version", time.Millisecond*50, nil)

	collector.RecordFileOperation("read", 1024, nil)
	collector.RecordFileOperation("write", 2048, nil)

	collector.RecordSchemaGeneration(100, time.Millisecond*500, nil)

	// Record cache stats
	mockStats := &MockCacheStatsProvider{
		CacheStats: CacheStats{
			Hits:      100,
			Misses:    20,
			Size:      10240,
			MaxSize:   102400,
			Evictions: 5,
			Keys:      15,
		},
	}
	collector.RecordHelmCacheStats(mockStats)

	// Stop the collector
	err = collector.Stop(ctx)
	assert.NoError(t, err)
}

// TestEndToEndWithMocks demonstrates using mocks for testing
func TestEndToEndWithMocks(t *testing.T) {
	ctx := context.Background()

	// Create mock provider
	mockProvider := NewMockProvider()

	// Set up custom behavior
	traceCount := 0
	mockProvider.TracerFunc = func(name string) trace.Tracer {
		traceCount++
		return trace.NewNoopTracerProvider().Tracer(name)
	}

	// Create mock metrics collector
	mockCollector := NewMockMetricsCollector()

	// Simulate application usage
	_ = mockProvider.Tracer("app")
	assert.Equal(t, 1, traceCount)

	_ = mockProvider.Meter("app")
	assert.Equal(t, 1, mockProvider.MeterCallCount)

	// Record metrics
	mockCollector.RecordCommandExecution(ctx, "test", time.Second, nil)
	assert.Equal(t, 1, mockCollector.RecordCommandExecutionCallCount)
	assert.Len(t, mockCollector.RecordedCommands, 1)

	// Verify the recorded data
	cmd := mockCollector.RecordedCommands[0]
	assert.Equal(t, "test", cmd.Command)
	assert.Equal(t, time.Second, cmd.Duration)
	assert.Nil(t, cmd.Error)

	// Test shutdown
	err := mockProvider.Shutdown(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, mockProvider.ShutdownCallCount)
}

// TestMixedImplementations tests using real and mock implementations together
func TestMixedImplementations(t *testing.T) {
	ctx := context.Background()

	// Use real provider with mock collector
	provider, err := NewProvider(ctx, ProviderOptions{
		Type: ProviderTypeNoop,
	})
	require.NoError(t, err)

	mockCollector := NewMockMetricsCollector()

	// Simulate telemetry operations
	tracer := provider.Tracer("mixed-test")
	ctx, span := tracer.Start(ctx, "operation")

	// Record metrics with mock
	mockCollector.RecordCommandExecution(ctx, "mixed", time.Millisecond*100, nil)

	span.End()

	// Verify mock was called
	assert.Equal(t, 1, mockCollector.RecordCommandExecutionCallCount)

	// Real provider shutdown
	err = provider.Shutdown(ctx)
	assert.NoError(t, err)
}

// TestFactoryPatternUsage demonstrates the factory pattern benefits
func TestFactoryPatternUsage(t *testing.T) {
	ctx := context.Background()

	// Test switching providers based on config
	configs := []struct {
		name     string
		provType ProviderType
		config   *config.TelemetryConfig
	}{
		{
			name:     "Development - No-op",
			provType: ProviderTypeNoop,
			config:   nil,
		},
		{
			name:     "Testing - OpenTelemetry with stdout",
			provType: ProviderTypeOpenTelemetry,
			config: &config.TelemetryConfig{
				Enabled:      true,
				ExporterType: "stdout",
			},
		},
		{
			name:     "Production - OpenTelemetry with OTLP",
			provType: ProviderTypeOpenTelemetry,
			config: &config.TelemetryConfig{
				Enabled:      true,
				ExporterType: "otlp",
				OTLPEndpoint: "localhost:4317",
			},
		},
	}

	for _, tc := range configs {
		t.Run(tc.name, func(t *testing.T) {
			provider, err := NewProvider(ctx, ProviderOptions{
				Type:   tc.provType,
				Config: tc.config,
			})

			// All configurations should work
			assert.NoError(t, err)
			assert.NotNil(t, provider)

			// Basic operations should work
			tracer := provider.Tracer("test")
			assert.NotNil(t, tracer)

			// Cleanup
			_ = provider.Shutdown(ctx)
		})
	}
}
