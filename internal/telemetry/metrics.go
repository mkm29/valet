package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/mkm29/valet/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// MetricsServer handles Prometheus metrics collection and exposure
type MetricsServer struct {
	server *http.Server
	logger *zap.Logger
	mu     sync.RWMutex

	// Helm cache metrics
	helmCacheHits       prometheus.Counter
	helmCacheMisses     prometheus.Counter
	helmCacheEvictions  prometheus.Counter
	helmCacheSize       prometheus.Gauge
	helmCacheEntries    prometheus.Gauge
	helmCacheHitRate    prometheus.Gauge
	helmMetadataHits    prometheus.Counter
	helmMetadataMisses  prometheus.Counter
	helmMetadataEntries prometheus.Gauge
	helmMetadataHitRate prometheus.Gauge

	// Command metrics
	commandExecutions *prometheus.CounterVec
	commandDuration   *prometheus.HistogramVec
	commandErrors     *prometheus.CounterVec

	// Schema generation metrics
	schemaGenerations      prometheus.Counter
	schemaGenerationErrors prometheus.Counter
	schemaFields           prometheus.Histogram
	schemaGenerationTime   prometheus.Histogram

	// File operation metrics
	fileReads       prometheus.Counter
	fileWrites      prometheus.Counter
	fileReadErrors  prometheus.Counter
	fileWriteErrors prometheus.Counter
	fileSize        prometheus.Histogram
}

// MetricsConfig holds configuration for the metrics server
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	Path    string `yaml:"path"`
}

// NewMetricsConfig returns a default metrics configuration
func NewMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		Enabled: false,
		Port:    9090,
		Path:    "/metrics",
	}
}

// NewMetricsServer creates a new metrics server instance
func NewMetricsServer(config *MetricsConfig, logger *zap.Logger) *MetricsServer {
	if logger == nil {
		logger = zap.L().Named("metrics")
	}

	m := &MetricsServer{
		logger: logger,

		// Initialize Helm cache metrics
		helmCacheHits: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "valet",
			Subsystem: "helm_cache",
			Name:      "hits_total",
			Help:      "Total number of Helm chart cache hits",
		}),
		helmCacheMisses: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "valet",
			Subsystem: "helm_cache",
			Name:      "misses_total",
			Help:      "Total number of Helm chart cache misses",
		}),
		helmCacheEvictions: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "valet",
			Subsystem: "helm_cache",
			Name:      "evictions_total",
			Help:      "Total number of Helm chart cache evictions",
		}),
		helmCacheSize: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "valet",
			Subsystem: "helm_cache",
			Name:      "size_bytes",
			Help:      "Current size of the Helm chart cache in bytes",
		}),
		helmCacheEntries: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "valet",
			Subsystem: "helm_cache",
			Name:      "entries",
			Help:      "Current number of entries in the Helm chart cache",
		}),
		helmCacheHitRate: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "valet",
			Subsystem: "helm_cache",
			Name:      "hit_rate",
			Help:      "Helm chart cache hit rate (0-100)",
		}),
		helmMetadataHits: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "valet",
			Subsystem: "helm_metadata_cache",
			Name:      "hits_total",
			Help:      "Total number of Helm metadata cache hits",
		}),
		helmMetadataMisses: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "valet",
			Subsystem: "helm_metadata_cache",
			Name:      "misses_total",
			Help:      "Total number of Helm metadata cache misses",
		}),
		helmMetadataEntries: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "valet",
			Subsystem: "helm_metadata_cache",
			Name:      "entries",
			Help:      "Current number of entries in the Helm metadata cache",
		}),
		helmMetadataHitRate: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "valet",
			Subsystem: "helm_metadata_cache",
			Name:      "hit_rate",
			Help:      "Helm metadata cache hit rate (0-100)",
		}),

		// Initialize command metrics
		commandExecutions: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "valet",
			Subsystem: "command",
			Name:      "executions_total",
			Help:      "Total number of command executions",
		}, []string{"command"}),
		commandDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "valet",
			Subsystem: "command",
			Name:      "duration_seconds",
			Help:      "Command execution duration in seconds",
			Buckets:   prometheus.DefBuckets,
		}, []string{"command"}),
		commandErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "valet",
			Subsystem: "command",
			Name:      "errors_total",
			Help:      "Total number of command errors",
		}, []string{"command"}),

		// Initialize schema generation metrics
		schemaGenerations: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "valet",
			Name:      "schema_generations_total",
			Help:      "Total number of schema generations",
		}),
		schemaGenerationErrors: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "valet",
			Name:      "schema_generation_errors_total",
			Help:      "Total number of schema generation errors",
		}),
		schemaFields: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "valet",
			Name:      "schema_fields",
			Help:      "Number of fields in generated schemas",
			Buckets:   []float64{10, 25, 50, 100, 250, 500, 1000},
		}),
		schemaGenerationTime: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "valet",
			Name:      "schema_generation_duration_seconds",
			Help:      "Schema generation duration in seconds",
			Buckets:   prometheus.DefBuckets,
		}),

		// Initialize file operation metrics
		fileReads: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "valet",
			Subsystem: "file",
			Name:      "reads_total",
			Help:      "Total number of file read operations",
		}),
		fileWrites: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "valet",
			Subsystem: "file",
			Name:      "writes_total",
			Help:      "Total number of file write operations",
		}),
		fileReadErrors: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "valet",
			Subsystem: "file",
			Name:      "read_errors_total",
			Help:      "Total number of file read errors",
		}),
		fileWriteErrors: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "valet",
			Subsystem: "file",
			Name:      "write_errors_total",
			Help:      "Total number of file write errors",
		}),
		fileSize: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "valet",
			Subsystem: "file",
			Name:      "size_bytes",
			Help:      "Size of files in bytes",
			Buckets:   []float64{1024, 10240, 102400, 1048576, 10485760}, // 1KB, 10KB, 100KB, 1MB, 10MB
		}),
	}

	// Create HTTP server for metrics
	mux := http.NewServeMux()
	mux.Handle(config.Path, promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	m.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: mux,
	}

	return m
}

// Start starts the metrics HTTP server
func (m *MetricsServer) Start(ctx context.Context) error {
	m.logger.Info("Starting metrics server",
		zap.String("address", m.server.Addr),
	)

	errCh := make(chan error, 1)
	go func() {
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("metrics server error: %w", err)
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		return m.Shutdown(context.Background())
	case err := <-errCh:
		return err
	case <-time.After(100 * time.Millisecond):
		// Give server time to start and fail fast if port is already in use
		select {
		case err := <-errCh:
			return err
		default:
			m.logger.Info("Metrics server started successfully")
			return nil
		}
	}
}

// Shutdown gracefully shuts down the metrics server
func (m *MetricsServer) Shutdown(ctx context.Context) error {
	m.logger.Info("Shutting down metrics server")
	return m.server.Shutdown(ctx)
}

// UpdateHelmCacheStats updates all Helm cache metrics from provided stats
func (m *MetricsServer) UpdateHelmCacheStats(stats interface{}) {
	// Type assertion to handle both HelmCacheStats and the helm package's stats type
	var helmStats HelmCacheStats

	switch v := stats.(type) {
	case HelmCacheStats:
		helmStats = v
	default:
		// Use reflection to extract fields from helm.CacheStats
		// This allows us to avoid circular dependencies
		rv := reflect.ValueOf(stats)
		if rv.Kind() == reflect.Struct {
			// Extract fields by name
			helmStats = HelmCacheStats{
				Entries:         int(utils.GetFieldInt(rv, "Entries")),
				CurrentSize:     utils.GetFieldInt64(rv, "CurrentSize"),
				MaxSize:         utils.GetFieldInt64(rv, "MaxSize"),
				MaxEntries:      int(utils.GetFieldInt(rv, "MaxEntries")),
				Hits:            utils.GetFieldInt64(rv, "Hits"),
				Misses:          utils.GetFieldInt64(rv, "Misses"),
				Evictions:       utils.GetFieldInt64(rv, "Evictions"),
				HitRate:         utils.GetFieldFloat64(rv, "HitRate"),
				UsagePercent:    utils.GetFieldFloat64(rv, "UsagePercent"),
				MetadataEntries: int(utils.GetFieldInt(rv, "MetadataEntries")),
				MetadataHits:    utils.GetFieldInt64(rv, "MetadataHits"),
				MetadataMisses:  utils.GetFieldInt64(rv, "MetadataMisses"),
				MetadataHitRate: utils.GetFieldFloat64(rv, "MetadataHitRate"),
			}
		} else {
			m.logger.Warn("Invalid stats type received", zap.String("type", fmt.Sprintf("%T", v)))
			return
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Update counters (these are cumulative, so we set them to the total values)
	// Prometheus counters can only increase, so we need to track deltas
	m.helmCacheHits.Add(float64(helmStats.Hits - m.getLastHits()))
	m.helmCacheMisses.Add(float64(helmStats.Misses - m.getLastMisses()))
	m.helmCacheEvictions.Add(float64(helmStats.Evictions - m.getLastEvictions()))

	// Update gauges (these can go up or down)
	m.helmCacheSize.Set(float64(helmStats.CurrentSize))
	m.helmCacheEntries.Set(float64(helmStats.Entries))
	m.helmCacheHitRate.Set(helmStats.HitRate)

	// Update metadata cache metrics
	m.helmMetadataHits.Add(float64(helmStats.MetadataHits - m.getLastMetadataHits()))
	m.helmMetadataMisses.Add(float64(helmStats.MetadataMisses - m.getLastMetadataMisses()))
	m.helmMetadataEntries.Set(float64(helmStats.MetadataEntries))
	m.helmMetadataHitRate.Set(helmStats.MetadataHitRate)

	// Store last values for delta calculation
	m.storeLastValues(helmStats)
}

// Helper functions moved to utils package

// HelmCacheStats represents cache statistics from the Helm package
type HelmCacheStats struct {
	Entries         int
	CurrentSize     int64
	MaxSize         int64
	MaxEntries      int
	Hits            int64
	Misses          int64
	Evictions       int64
	HitRate         float64
	UsagePercent    float64
	MetadataEntries int
	MetadataHits    int64
	MetadataMisses  int64
	MetadataHitRate float64
}

// Command metrics methods

// RecordCommandExecution records a command execution
func (m *MetricsServer) RecordCommandExecution(command string, duration time.Duration, err error) {
	m.commandExecutions.WithLabelValues(command).Inc()
	m.commandDuration.WithLabelValues(command).Observe(duration.Seconds())
	if err != nil {
		m.commandErrors.WithLabelValues(command).Inc()
	}
}

// Schema generation metrics methods

// RecordSchemaGeneration records schema generation metrics
func (m *MetricsServer) RecordSchemaGeneration(fieldCount int, duration time.Duration, err error) {
	m.schemaGenerations.Inc()
	m.schemaFields.Observe(float64(fieldCount))
	m.schemaGenerationTime.Observe(duration.Seconds())
	if err != nil {
		m.schemaGenerationErrors.Inc()
	}
}

// File operation metrics methods

// RecordFileRead records a file read operation
func (m *MetricsServer) RecordFileRead(size int64, err error) {
	m.fileReads.Inc()
	if err != nil {
		m.fileReadErrors.Inc()
	} else {
		m.fileSize.Observe(float64(size))
	}
}

// RecordFileWrite records a file write operation
func (m *MetricsServer) RecordFileWrite(size int64, err error) {
	m.fileWrites.Inc()
	if err != nil {
		m.fileWriteErrors.Inc()
	} else {
		m.fileSize.Observe(float64(size))
	}
}

// Internal state tracking for delta calculations
var (
	lastHits           int64
	lastMisses         int64
	lastEvictions      int64
	lastMetadataHits   int64
	lastMetadataMisses int64
	stateMu            sync.Mutex
)

func (m *MetricsServer) getLastHits() int64 {
	stateMu.Lock()
	defer stateMu.Unlock()
	return lastHits
}

func (m *MetricsServer) getLastMisses() int64 {
	stateMu.Lock()
	defer stateMu.Unlock()
	return lastMisses
}

func (m *MetricsServer) getLastEvictions() int64 {
	stateMu.Lock()
	defer stateMu.Unlock()
	return lastEvictions
}

func (m *MetricsServer) getLastMetadataHits() int64 {
	stateMu.Lock()
	defer stateMu.Unlock()
	return lastMetadataHits
}

func (m *MetricsServer) getLastMetadataMisses() int64 {
	stateMu.Lock()
	defer stateMu.Unlock()
	return lastMetadataMisses
}

func (m *MetricsServer) storeLastValues(stats HelmCacheStats) {
	stateMu.Lock()
	defer stateMu.Unlock()
	lastHits = stats.Hits
	lastMisses = stats.Misses
	lastEvictions = stats.Evictions
	lastMetadataHits = stats.MetadataHits
	lastMetadataMisses = stats.MetadataMisses
}
