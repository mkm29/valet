package telemetry

import (
	"context"
	"net/http"
	"time"
)

// MetricsCollector defines the interface for metrics collection
type MetricsCollector interface {
	// RecordCommandExecution records command execution metrics
	RecordCommandExecution(ctx context.Context, command string, duration time.Duration, err error)

	// RecordFileOperation records file operation metrics
	RecordFileOperation(operation string, size int64, err error)

	// RecordSchemaGeneration records schema generation metrics
	RecordSchemaGeneration(fields int, duration time.Duration, err error)

	// RecordHelmCacheStats records Helm cache statistics
	RecordHelmCacheStats(stats CacheStatsProvider)

	// Start starts the metrics collection (if needed)
	Start(ctx context.Context) error

	// Stop stops the metrics collection
	Stop(ctx context.Context) error
}

// MetricsExporter defines how metrics are exposed
type MetricsExporter interface {
	// Handler returns the HTTP handler for metrics endpoint
	Handler() http.Handler

	// Endpoint returns the endpoint path for metrics
	Endpoint() string
}

// CacheStatsProvider is defined in metrics.go

// MetricsCollectorType represents the type of metrics collector
type MetricsCollectorType string

const (
	MetricsCollectorTypePrometheus MetricsCollectorType = "prometheus"
	MetricsCollectorTypeNoop       MetricsCollectorType = "noop"
	// Future collectors can be added here
	// MetricsCollectorTypeStatsD MetricsCollectorType = "statsd"
	// MetricsCollectorTypeCloudWatch MetricsCollectorType = "cloudwatch"
)

// MetricsCollectorOptions configures the metrics collector
type MetricsCollectorOptions struct {
	Type   MetricsCollectorType
	Config interface{} // Collector-specific configuration
	Logger interface{} // Can be logr.Logger or *slog.Logger
}

// NewMetricsCollector creates a new metrics collector based on options
func NewMetricsCollector(opts MetricsCollectorOptions) (MetricsCollector, error) {
	switch opts.Type {
	case MetricsCollectorTypePrometheus:
		return newPrometheusCollector(opts)
	case MetricsCollectorTypeNoop:
		return newNoopMetricsCollector(), nil
	default:
		// Default to no-op collector
		return newNoopMetricsCollector(), nil
	}
}

// MetricsServer combines collector and exporter functionality
type MetricsServerInterface interface {
	MetricsCollector
	MetricsExporter
}
