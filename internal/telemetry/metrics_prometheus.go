package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	"github.com/mkm29/valet/internal/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// prometheusCollector implements MetricsCollector using Prometheus
type prometheusCollector struct {
	server *MetricsServer
}

// newPrometheusCollector creates a new Prometheus metrics collector
func newPrometheusCollector(opts MetricsCollectorOptions) (MetricsCollector, error) {
	cfg, ok := opts.Config.(*config.MetricsConfig)
	if !ok {
		// If no config provided, use defaults
		cfg = &config.MetricsConfig{
			Enabled: true,
			Port:    2112,
			Path:    "/metrics",
		}
	}

	// Determine logger type and create appropriate slog.Logger
	var slogger *slog.Logger
	switch l := opts.Logger.(type) {
	case *slog.Logger:
		slogger = l
	case logr.Logger:
		// Convert logr to slog for MetricsServer
		slogger = slog.Default().With("component", "metrics")
	default:
		// Create a default logger if none provided
		slogger = slog.Default().With("component", "metrics")
	}

	// Reuse the existing MetricsServer which already has all the Prometheus setup
	server := NewMetricsServer(cfg, slogger)

	return &prometheusCollector{
		server: server,
	}, nil
}

// RecordCommandExecution records command execution metrics
func (c *prometheusCollector) RecordCommandExecution(ctx context.Context, command string, duration time.Duration, err error) {
	c.server.RecordCommandExecution(ctx, command, duration, err)
}

// RecordFileOperation records file operation metrics
func (c *prometheusCollector) RecordFileOperation(operation string, size int64, err error) {
	// The MetricsServer has separate methods for read and write
	switch operation {
	case "read":
		c.server.RecordFileRead(context.Background(), size, err)
	case "write":
		c.server.RecordFileWrite(context.Background(), size, err)
	default:
		// For other operations, just record as read
		c.server.RecordFileRead(context.Background(), size, err)
	}
}

// RecordSchemaGeneration records schema generation metrics
func (c *prometheusCollector) RecordSchemaGeneration(fields int, duration time.Duration, err error) {
	c.server.RecordSchemaGeneration(context.Background(), fields, duration, err)
}

// RecordHelmCacheStats records Helm cache statistics
func (c *prometheusCollector) RecordHelmCacheStats(stats CacheStatsProvider) {
	c.server.UpdateHelmCacheStats(stats)
}

// Start starts the metrics server
func (c *prometheusCollector) Start(ctx context.Context) error {
	return c.server.Start(ctx)
}

// Stop stops the metrics server
func (c *prometheusCollector) Stop(ctx context.Context) error {
	return c.server.Shutdown(ctx)
}

// prometheusMetricsServer extends prometheusCollector with exporter functionality
type prometheusMetricsServer struct {
	*prometheusCollector
}

// Handler returns the HTTP handler for metrics
func (s *prometheusMetricsServer) Handler() http.Handler {
	return promhttp.Handler()
}

// Endpoint returns the metrics endpoint
func (s *prometheusMetricsServer) Endpoint() string {
	if s.server != nil && s.server.config != nil {
		return s.server.config.Path
	}
	return "/metrics"
}

// NewPrometheusMetricsServer creates a combined collector and exporter
func NewPrometheusMetricsServer(cfg *config.MetricsConfig, logger logr.Logger) (MetricsServerInterface, error) {
	collector, err := newPrometheusCollector(MetricsCollectorOptions{
		Type:   MetricsCollectorTypePrometheus,
		Config: cfg,
		Logger: logger,
	})
	if err != nil {
		return nil, err
	}

	promCollector, ok := collector.(*prometheusCollector)
	if !ok {
		return nil, fmt.Errorf("unexpected collector type")
	}

	return &prometheusMetricsServer{
		prometheusCollector: promCollector,
	}, nil
}
