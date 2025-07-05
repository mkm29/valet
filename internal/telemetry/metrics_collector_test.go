package telemetry

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mkm29/valet/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetricsCollector(t *testing.T) {
	tests := []struct {
		name         string
		opts         MetricsCollectorOptions
		expectError  bool
		expectedType string
	}{
		{
			name: "Prometheus collector",
			opts: MetricsCollectorOptions{
				Type: MetricsCollectorTypePrometheus,
				Config: &config.MetricsConfig{
					Enabled: true,
					Port:    2112,
				},
			},
			expectError:  false,
			expectedType: "*telemetry.prometheusCollector",
		},
		{
			name: "No-op collector",
			opts: MetricsCollectorOptions{
				Type: MetricsCollectorTypeNoop,
			},
			expectError:  false,
			expectedType: "*telemetry.noopMetricsCollector",
		},
		{
			name: "Default to no-op collector",
			opts: MetricsCollectorOptions{
				Type: "unknown",
			},
			expectError:  false,
			expectedType: "*telemetry.noopMetricsCollector",
		},
		{
			name: "Prometheus with nil config uses defaults",
			opts: MetricsCollectorOptions{
				Type:   MetricsCollectorTypePrometheus,
				Config: nil,
			},
			expectError:  false,
			expectedType: "*telemetry.prometheusCollector",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector, err := NewMetricsCollector(tt.opts)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, collector)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, collector)

				// Check the actual type
				actualType := getTypeName(collector)
				assert.Equal(t, tt.expectedType, actualType)

				// Basic functionality test
				ctx := context.Background()
				collector.RecordCommandExecution(ctx, "test", time.Second, nil)
				collector.RecordFileOperation("read", 1024, nil)
				collector.RecordSchemaGeneration(10, time.Millisecond*100, nil)

				// For Prometheus collector, stop the server if started
				if pc, ok := collector.(*prometheusCollector); ok && pc.server != nil {
					_ = collector.Stop(ctx)
				}
			}
		})
	}
}

func TestNoopMetricsCollector(t *testing.T) {
	collector := newNoopMetricsCollector()
	assert.NotNil(t, collector)

	ctx := context.Background()

	// All operations should be no-ops and not panic
	collector.RecordCommandExecution(ctx, "test", time.Second, nil)
	collector.RecordFileOperation("write", 2048, nil)
	collector.RecordSchemaGeneration(20, time.Millisecond*200, nil)

	// Create a mock cache stats provider
	mockStats := NewMockCacheStatsProvider()
	collector.RecordHelmCacheStats(mockStats)

	// Start and stop should return nil
	err := collector.Start(ctx)
	assert.NoError(t, err)

	err = collector.Stop(ctx)
	assert.NoError(t, err)
}

func TestMockMetricsCollector(t *testing.T) {
	mock := NewMockMetricsCollector()
	assert.NotNil(t, mock)

	ctx := context.Background()
	testErr := errors.New("test error")

	// Test RecordCommandExecution
	mock.RecordCommandExecution(ctx, "generate", time.Second*2, testErr)
	assert.Equal(t, 1, mock.RecordCommandExecutionCallCount)
	assert.Len(t, mock.RecordedCommands, 1)
	assert.Equal(t, "generate", mock.RecordedCommands[0].Command)
	assert.Equal(t, time.Second*2, mock.RecordedCommands[0].Duration)
	assert.Equal(t, testErr, mock.RecordedCommands[0].Error)

	// Test RecordFileOperation
	mock.RecordFileOperation("read", 4096, nil)
	assert.Equal(t, 1, mock.RecordFileOperationCallCount)
	assert.Len(t, mock.RecordedFileOperations, 1)
	assert.Equal(t, "read", mock.RecordedFileOperations[0].Operation)
	assert.Equal(t, int64(4096), mock.RecordedFileOperations[0].Size)
	assert.Nil(t, mock.RecordedFileOperations[0].Error)

	// Test RecordSchemaGeneration
	mock.RecordSchemaGeneration(50, time.Millisecond*500, nil)
	assert.Equal(t, 1, mock.RecordSchemaGenerationCallCount)
	assert.Len(t, mock.RecordedSchemaGenerations, 1)
	assert.Equal(t, 50, mock.RecordedSchemaGenerations[0].Fields)
	assert.Equal(t, time.Millisecond*500, mock.RecordedSchemaGenerations[0].Duration)

	// Test RecordHelmCacheStats
	mockStats := NewMockCacheStatsProvider()
	mock.RecordHelmCacheStats(mockStats)
	assert.Equal(t, 1, mock.RecordHelmCacheStatsCallCount)
	assert.Len(t, mock.RecordedCacheStats, 1)

	// Test Start
	err := mock.Start(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, mock.StartCallCount)

	// Test Stop
	err = mock.Stop(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, mock.StopCallCount)

	// Test custom error behavior
	mock.StartFunc = nil // Clear function to use error field
	mock.StartError = errors.New("start failed")
	err = mock.Start(ctx)
	assert.Error(t, err)
	assert.Equal(t, "start failed", err.Error())

	// Test Reset
	mock.Reset()
	assert.Equal(t, 0, mock.RecordCommandExecutionCallCount)
	assert.Empty(t, mock.RecordedCommands)
	assert.Equal(t, 0, mock.RecordFileOperationCallCount)
	assert.Empty(t, mock.RecordedFileOperations)
}

func TestMockMetricsExporter(t *testing.T) {
	mock := NewMockMetricsExporter()
	assert.NotNil(t, mock)

	// Test Handler
	handler := mock.Handler()
	assert.NotNil(t, handler)
	assert.Equal(t, 1, mock.HandlerCallCount)

	// Test that the default handler works
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "mock metrics", w.Body.String())

	// Test Endpoint
	endpoint := mock.Endpoint()
	assert.Equal(t, "/metrics", endpoint)
	assert.Equal(t, 1, mock.EndpointCallCount)

	// Test custom endpoint
	mock.EndpointFunc = func() string {
		return "/custom/metrics"
	}
	endpoint = mock.Endpoint()
	assert.Equal(t, "/custom/metrics", endpoint)
	assert.Equal(t, "/custom/metrics", mock.EndpointResult)
}

func TestMockMetricsServer(t *testing.T) {
	mock := NewMockMetricsServer()
	assert.NotNil(t, mock)

	ctx := context.Background()

	// Test that it has both collector and exporter functionality
	mock.RecordCommandExecution(ctx, "test", time.Second, nil)
	assert.Equal(t, 1, mock.MockMetricsCollector.RecordCommandExecutionCallCount)

	handler := mock.Handler()
	assert.NotNil(t, handler)
	assert.Equal(t, 1, mock.MockMetricsExporter.HandlerCallCount)

	// Test Reset
	mock.Reset()
	assert.Equal(t, 0, mock.MockMetricsCollector.RecordCommandExecutionCallCount)
	assert.Equal(t, 0, mock.MockMetricsExporter.HandlerCallCount)
}

func TestMetricsCollectorInterfaceCompliance(t *testing.T) {
	// Ensure all implementations satisfy the interfaces
	var _ MetricsCollector = (*prometheusCollector)(nil)
	var _ MetricsCollector = (*noopMetricsCollector)(nil)
	var _ MetricsCollector = (*MockMetricsCollector)(nil)

	var _ MetricsExporter = (*MockMetricsExporter)(nil)
	var _ MetricsServerInterface = (*MockMetricsServer)(nil)
}

func TestPrometheusCollectorIntegration(t *testing.T) {
	// This test demonstrates the Prometheus collector working with the metrics server
	ctx := context.Background()

	// Create a Prometheus collector
	collector, err := NewMetricsCollector(MetricsCollectorOptions{
		Type: MetricsCollectorTypePrometheus,
		Config: &config.MetricsConfig{
			Enabled: true,
			Port:    0, // Use random port
		},
	})
	require.NoError(t, err)
	require.NotNil(t, collector)

	// Record some metrics
	collector.RecordCommandExecution(ctx, "generate", time.Second, nil)
	collector.RecordCommandExecution(ctx, "generate", time.Millisecond*500, errors.New("failed"))
	collector.RecordFileOperation("read", 1024, nil)
	collector.RecordFileOperation("write", 2048, errors.New("permission denied"))
	collector.RecordSchemaGeneration(100, time.Millisecond*250, nil)

	// Record cache stats
	mockStats := NewMockCacheStatsProvider()
	collector.RecordHelmCacheStats(mockStats)

	// The Prometheus collector wraps a MetricsServer which starts automatically
	// We just need to stop it when done
	defer func() {
		err := collector.Stop(ctx)
		assert.NoError(t, err)
	}()
}

func TestMetricsCollectorFactoryPattern(t *testing.T) {
	// Test that the factory correctly creates collectors based on type
	testCases := []struct {
		collectorType MetricsCollectorType
		expectedType  string
	}{
		{MetricsCollectorTypePrometheus, "*telemetry.prometheusCollector"},
		{MetricsCollectorTypeNoop, "*telemetry.noopMetricsCollector"},
		{"invalid-type", "*telemetry.noopMetricsCollector"}, // Should default to no-op
	}

	for _, tc := range testCases {
		t.Run(string(tc.collectorType), func(t *testing.T) {
			collector, err := NewMetricsCollector(MetricsCollectorOptions{
				Type: tc.collectorType,
			})

			assert.NoError(t, err)
			assert.NotNil(t, collector)
			assert.Equal(t, tc.expectedType, getTypeName(collector))

			// Cleanup
			ctx := context.Background()
			_ = collector.Stop(ctx)
		})
	}
}

func TestCacheStatsProvider(t *testing.T) {
	// Test the mock cache stats provider
	mock := NewMockCacheStatsProvider()
	assert.NotNil(t, mock)

	// Test default values
	assert.Equal(t, 10, mock.GetEntries())
	assert.Equal(t, int64(1024), mock.GetCurrentSize())
	assert.Equal(t, int64(10240), mock.GetMaxSize())
	assert.Equal(t, 1000, mock.GetMaxEntries())
	assert.Equal(t, int64(10), mock.GetHits())
	assert.Equal(t, int64(2), mock.GetMisses())
	assert.Equal(t, int64(1), mock.GetEvictions())

	// Test calculated values
	hitRate := mock.GetHitRate()
	assert.InDelta(t, 83.33, hitRate, 0.01) // 10/(10+2) * 100

	usagePercent := mock.GetUsagePercent()
	assert.Equal(t, float64(10), usagePercent) // 1024/10240 * 100

	// Test metadata methods
	assert.Equal(t, 0, mock.GetMetadataEntries())
	assert.Equal(t, int64(0), mock.GetMetadataHits())
	assert.Equal(t, int64(0), mock.GetMetadataMisses())
	assert.Equal(t, float64(0), mock.GetMetadataHitRate())

	// Test that methods work
	stats := CacheStats{
		Hits:      mock.GetHits(),
		Misses:    mock.GetMisses(),
		Size:      mock.GetCurrentSize(),
		MaxSize:   mock.GetMaxSize(),
		Evictions: mock.GetEvictions(),
		Keys:      mock.GetEntries(),
	}
	assert.Equal(t, int64(10), stats.Hits)
	assert.Equal(t, int64(2), stats.Misses)
	assert.Equal(t, int64(1024), stats.Size)
}
