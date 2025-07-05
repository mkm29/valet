package telemetry

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// MockMetricsCollector is a mock implementation of MetricsCollector for testing
type MockMetricsCollector struct {
	mu sync.Mutex

	// RecordCommandExecution behavior
	RecordCommandExecutionFunc      func(ctx context.Context, command string, duration time.Duration, err error)
	RecordCommandExecutionCallCount int
	RecordedCommands                []struct {
		Command  string
		Duration time.Duration
		Error    error
	}

	// RecordFileOperation behavior
	RecordFileOperationFunc      func(operation string, size int64, err error)
	RecordFileOperationCallCount int
	RecordedFileOperations       []struct {
		Operation string
		Size      int64
		Error     error
	}

	// RecordSchemaGeneration behavior
	RecordSchemaGenerationFunc      func(fields int, duration time.Duration, err error)
	RecordSchemaGenerationCallCount int
	RecordedSchemaGenerations       []struct {
		Fields   int
		Duration time.Duration
		Error    error
	}

	// RecordHelmCacheStats behavior
	RecordHelmCacheStatsFunc      func(stats CacheStatsProvider)
	RecordHelmCacheStatsCallCount int
	RecordedCacheStats            []CacheStatsProvider

	// Start behavior
	StartFunc      func(ctx context.Context) error
	StartCallCount int
	StartError     error

	// Stop behavior
	StopFunc      func(ctx context.Context) error
	StopCallCount int
	StopError     error
}

// NewMockMetricsCollector creates a new mock metrics collector
func NewMockMetricsCollector() *MockMetricsCollector {
	return &MockMetricsCollector{
		RecordCommandExecutionFunc: func(ctx context.Context, command string, duration time.Duration, err error) {},
		RecordFileOperationFunc:    func(operation string, size int64, err error) {},
		RecordSchemaGenerationFunc: func(fields int, duration time.Duration, err error) {},
		RecordHelmCacheStatsFunc:   func(stats CacheStatsProvider) {},
		StartFunc:                  func(ctx context.Context) error { return nil },
		StopFunc:                   func(ctx context.Context) error { return nil },
	}
}

// RecordCommandExecution records command execution metrics
func (m *MockMetricsCollector) RecordCommandExecution(ctx context.Context, command string, duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RecordCommandExecutionCallCount++
	m.RecordedCommands = append(m.RecordedCommands, struct {
		Command  string
		Duration time.Duration
		Error    error
	}{
		Command:  command,
		Duration: duration,
		Error:    err,
	})

	if m.RecordCommandExecutionFunc != nil {
		m.RecordCommandExecutionFunc(ctx, command, duration, err)
	}
}

// RecordFileOperation records file operation metrics
func (m *MockMetricsCollector) RecordFileOperation(operation string, size int64, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RecordFileOperationCallCount++
	m.RecordedFileOperations = append(m.RecordedFileOperations, struct {
		Operation string
		Size      int64
		Error     error
	}{
		Operation: operation,
		Size:      size,
		Error:     err,
	})

	if m.RecordFileOperationFunc != nil {
		m.RecordFileOperationFunc(operation, size, err)
	}
}

// RecordSchemaGeneration records schema generation metrics
func (m *MockMetricsCollector) RecordSchemaGeneration(fields int, duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RecordSchemaGenerationCallCount++
	m.RecordedSchemaGenerations = append(m.RecordedSchemaGenerations, struct {
		Fields   int
		Duration time.Duration
		Error    error
	}{
		Fields:   fields,
		Duration: duration,
		Error:    err,
	})

	if m.RecordSchemaGenerationFunc != nil {
		m.RecordSchemaGenerationFunc(fields, duration, err)
	}
}

// RecordHelmCacheStats records Helm cache statistics
func (m *MockMetricsCollector) RecordHelmCacheStats(stats CacheStatsProvider) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RecordHelmCacheStatsCallCount++
	m.RecordedCacheStats = append(m.RecordedCacheStats, stats)

	if m.RecordHelmCacheStatsFunc != nil {
		m.RecordHelmCacheStatsFunc(stats)
	}
}

// Start starts the metrics collection
func (m *MockMetricsCollector) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StartCallCount++

	if m.StartFunc != nil {
		return m.StartFunc(ctx)
	}
	return m.StartError
}

// Stop stops the metrics collection
func (m *MockMetricsCollector) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StopCallCount++

	if m.StopFunc != nil {
		return m.StopFunc(ctx)
	}
	return m.StopError
}

// Reset resets all call counts and recorded data
func (m *MockMetricsCollector) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RecordCommandExecutionCallCount = 0
	m.RecordedCommands = nil
	m.RecordFileOperationCallCount = 0
	m.RecordedFileOperations = nil
	m.RecordSchemaGenerationCallCount = 0
	m.RecordedSchemaGenerations = nil
	m.RecordHelmCacheStatsCallCount = 0
	m.RecordedCacheStats = nil
	m.StartCallCount = 0
	m.StopCallCount = 0
}

// MockMetricsExporter is a mock implementation of MetricsExporter for testing
type MockMetricsExporter struct {
	mu sync.Mutex

	HandlerFunc      func() http.Handler
	HandlerCallCount int
	HandlerResult    http.Handler

	EndpointFunc      func() string
	EndpointCallCount int
	EndpointResult    string
}

// NewMockMetricsExporter creates a new mock metrics exporter
func NewMockMetricsExporter() *MockMetricsExporter {
	return &MockMetricsExporter{
		HandlerFunc: func() http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("mock metrics"))
			})
		},
		EndpointFunc: func() string {
			return "/metrics"
		},
		EndpointResult: "/metrics",
	}
}

// Handler returns the HTTP handler for metrics endpoint
func (m *MockMetricsExporter) Handler() http.Handler {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.HandlerCallCount++

	if m.HandlerFunc != nil {
		result := m.HandlerFunc()
		m.HandlerResult = result
		return result
	}
	return m.HandlerResult
}

// Endpoint returns the endpoint path for metrics
func (m *MockMetricsExporter) Endpoint() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.EndpointCallCount++

	if m.EndpointFunc != nil {
		result := m.EndpointFunc()
		m.EndpointResult = result
		return result
	}
	return m.EndpointResult
}

// MockMetricsServer combines MockMetricsCollector and MockMetricsExporter
type MockMetricsServer struct {
	*MockMetricsCollector
	*MockMetricsExporter
}

// NewMockMetricsServer creates a new mock metrics server
func NewMockMetricsServer() *MockMetricsServer {
	return &MockMetricsServer{
		MockMetricsCollector: NewMockMetricsCollector(),
		MockMetricsExporter:  NewMockMetricsExporter(),
	}
}

// Reset resets both collector and exporter
func (m *MockMetricsServer) Reset() {
	m.MockMetricsCollector.Reset()
	// MockMetricsExporter doesn't have recorded data to reset, just counts
	m.MockMetricsExporter.mu.Lock()
	m.MockMetricsExporter.HandlerCallCount = 0
	m.MockMetricsExporter.EndpointCallCount = 0
	m.MockMetricsExporter.mu.Unlock()
}
