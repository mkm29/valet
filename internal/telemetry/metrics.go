package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// MetricsServer handles Prometheus metrics collection and exposure
type MetricsServer struct {
	server *http.Server
	logger *zap.Logger
	config *MetricsConfig
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

	// State tracking for delta calculations
	lastHits           int64
	lastMisses         int64
	lastEvictions      int64
	lastMetadataHits   int64
	lastMetadataMisses int64
	stateMu            sync.Mutex
}

// MetricsConfig holds configuration for the metrics server
type MetricsConfig struct {
	Enabled                bool          `yaml:"enabled"`
	Port                   int           `yaml:"port"`
	Path                   string        `yaml:"path"`
	HealthCheckMaxAttempts int           `yaml:"healthCheckMaxAttempts"`
	HealthCheckBackoff     time.Duration `yaml:"healthCheckBackoff"`
}

// NewMetricsConfig returns a default metrics configuration
func NewMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		Enabled:                false,
		Port:                   9090,
		Path:                   "/metrics",
		HealthCheckMaxAttempts: 10,
		HealthCheckBackoff:     50 * time.Millisecond,
	}
}

// NewMetricsServer creates a new metrics server instance
func NewMetricsServer(config *MetricsConfig, logger *zap.Logger) *MetricsServer {
	if logger == nil {
		logger = zap.L().Named("metrics")
	}

	m := &MetricsServer{
		logger: logger,
		config: config,

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

	// Wait for server startup confirmation or error
	if err := m.waitForServerStartup(ctx, errCh); err != nil {
		return err
	}

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		return m.Shutdown(context.Background())
	case err := <-errCh:
		return err
	}
}

// waitForServerStartup waits for the server to start up using health check polling
func (m *MetricsServer) waitForServerStartup(ctx context.Context, errCh chan error) error {
	maxAttempts := m.config.HealthCheckMaxAttempts
	backoff := m.config.HealthCheckBackoff

	for attempt := 0; attempt < maxAttempts; attempt++ {
		select {
		case err := <-errCh:
			return err
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Try to connect to the health endpoint
			client := &http.Client{Timeout: 100 * time.Millisecond}
			resp, err := client.Get(fmt.Sprintf("http://localhost%s/health", m.server.Addr))
			if err == nil {
				resp.Body.Close()
				m.logger.Info("Metrics server started successfully")
				return nil
			}

			// Wait before next attempt with exponential backoff
			time.Sleep(backoff)
			backoff = time.Duration(float64(backoff) * 1.5)
		}
	}

	return fmt.Errorf("metrics server failed to start after %d attempts", maxAttempts)
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
	case CacheStatsProvider:
		// Use interface methods for efficient access
		helmStats = HelmCacheStats{
			Entries:         v.GetEntries(),
			CurrentSize:     v.GetCurrentSize(),
			MaxSize:         v.GetMaxSize(),
			MaxEntries:      v.GetMaxEntries(),
			Hits:            v.GetHits(),
			Misses:          v.GetMisses(),
			Evictions:       v.GetEvictions(),
			HitRate:         v.GetHitRate(),
			UsagePercent:    v.GetUsagePercent(),
			MetadataEntries: v.GetMetadataEntries(),
			MetadataHits:    v.GetMetadataHits(),
			MetadataMisses:  v.GetMetadataMisses(),
			MetadataHitRate: v.GetMetadataHitRate(),
		}
	default:
		// Fall back to JSON marshaling/unmarshaling for compatibility
		data, err := json.Marshal(stats)
		if err != nil {
			m.logger.Warn("Failed to marshal stats", zap.Error(err))
			return
		}

		if err := json.Unmarshal(data, &helmStats); err != nil {
			m.logger.Warn("Failed to unmarshal stats", zap.Error(err))
			return
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Update counters with reset detection
	// Prometheus counters can only increase, so we need to track deltas
	hitsDelta := m.calculateDelta(helmStats.Hits, m.getLastHits())
	missesDelta := m.calculateDelta(helmStats.Misses, m.getLastMisses())
	evictionsDelta := m.calculateDelta(helmStats.Evictions, m.getLastEvictions())
	metadataHitsDelta := m.calculateDelta(helmStats.MetadataHits, m.getLastMetadataHits())
	metadataMissesDelta := m.calculateDelta(helmStats.MetadataMisses, m.getLastMetadataMisses())

	m.helmCacheHits.Add(float64(hitsDelta))
	m.helmCacheMisses.Add(float64(missesDelta))
	m.helmCacheEvictions.Add(float64(evictionsDelta))

	// Update gauges (these can go up or down)
	m.helmCacheSize.Set(float64(helmStats.CurrentSize))
	m.helmCacheEntries.Set(float64(helmStats.Entries))
	m.helmCacheHitRate.Set(helmStats.HitRate)

	// Update metadata cache metrics
	m.helmMetadataHits.Add(float64(metadataHitsDelta))
	m.helmMetadataMisses.Add(float64(metadataMissesDelta))
	m.helmMetadataEntries.Set(float64(helmStats.MetadataEntries))
	m.helmMetadataHitRate.Set(helmStats.MetadataHitRate)

	// Store last values for delta calculation
	m.storeLastValues(helmStats)
}

// CacheStatsProvider defines an interface for cache statistics
type CacheStatsProvider interface {
	GetEntries() int
	GetCurrentSize() int64
	GetMaxSize() int64
	GetMaxEntries() int
	GetHits() int64
	GetMisses() int64
	GetEvictions() int64
	GetHitRate() float64
	GetUsagePercent() float64
	GetMetadataEntries() int
	GetMetadataHits() int64
	GetMetadataMisses() int64
	GetMetadataHitRate() float64
}

// Helper functions moved to utils package

// HelmCacheStats represents cache statistics from the Helm package
type HelmCacheStats struct {
	Entries         int     `json:"entries"`
	CurrentSize     int64   `json:"currentSize"`
	MaxSize         int64   `json:"maxSize"`
	MaxEntries      int     `json:"maxEntries"`
	Hits            int64   `json:"hits"`
	Misses          int64   `json:"misses"`
	Evictions       int64   `json:"evictions"`
	HitRate         float64 `json:"hitRate"`
	UsagePercent    float64 `json:"usagePercent"`
	MetadataEntries int     `json:"metadataEntries"`
	MetadataHits    int64   `json:"metadataHits"`
	MetadataMisses  int64   `json:"metadataMisses"`
	MetadataHitRate float64 `json:"metadataHitRate"`
}

// Command metrics methods

// RecordCommandExecution records a command execution
func (m *MetricsServer) RecordCommandExecution(ctx context.Context, command string, duration time.Duration, err error) {
	m.commandExecutions.WithLabelValues(command).Inc()
	m.commandDuration.WithLabelValues(command).Observe(duration.Seconds())
	if err != nil {
		m.commandErrors.WithLabelValues(command).Inc()
	}

	// Add span attributes for distributed tracing correlation
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(
			attribute.String("metrics.command", command),
			attribute.Float64("metrics.duration_seconds", duration.Seconds()),
			attribute.Bool("metrics.has_error", err != nil),
		)
		if err != nil {
			span.SetAttributes(attribute.String("metrics.error", err.Error()))
		}
	}
}

// Schema generation metrics methods

// RecordSchemaGeneration records schema generation metrics
func (m *MetricsServer) RecordSchemaGeneration(ctx context.Context, fieldCount int, duration time.Duration, err error) {
	m.schemaGenerations.Inc()
	m.schemaFields.Observe(float64(fieldCount))
	m.schemaGenerationTime.Observe(duration.Seconds())
	if err != nil {
		m.schemaGenerationErrors.Inc()
	}

	// Add span attributes for distributed tracing correlation
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(
			attribute.Int("metrics.schema_fields", fieldCount),
			attribute.Float64("metrics.schema_duration_seconds", duration.Seconds()),
			attribute.Bool("metrics.schema_has_error", err != nil),
		)
		if err != nil {
			span.SetAttributes(attribute.String("metrics.schema_error", err.Error()))
		}
	}
}

// File operation metrics methods

// RecordFileRead records a file read operation
func (m *MetricsServer) RecordFileRead(ctx context.Context, size int64, err error) {
	m.fileReads.Inc()
	if err != nil {
		m.fileReadErrors.Inc()
	} else {
		m.fileSize.Observe(float64(size))
	}

	// Add span attributes for distributed tracing correlation
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(
			attribute.Int64("metrics.file_size_bytes", size),
			attribute.Bool("metrics.file_read_error", err != nil),
		)
		if err != nil {
			span.SetAttributes(attribute.String("metrics.file_read_error_msg", err.Error()))
		}
	}
}

// RecordFileWrite records a file write operation
func (m *MetricsServer) RecordFileWrite(ctx context.Context, size int64, err error) {
	m.fileWrites.Inc()
	if err != nil {
		m.fileWriteErrors.Inc()
	} else {
		m.fileSize.Observe(float64(size))
	}

	// Add span attributes for distributed tracing correlation
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(
			attribute.Int64("metrics.file_size_bytes", size),
			attribute.Bool("metrics.file_write_error", err != nil),
		)
		if err != nil {
			span.SetAttributes(attribute.String("metrics.file_write_error_msg", err.Error()))
		}
	}
}

// calculateDelta calculates the delta between current and last values,
// handling counter resets gracefully
func (m *MetricsServer) calculateDelta(current, last int64) int64 {
	if current < last {
		// Counter reset detected - use current value as the delta
		m.logger.Debug("Counter reset detected",
			zap.Int64("current", current),
			zap.Int64("last", last),
		)
		return current
	}
	return current - last
}

// Internal state tracking for delta calculations
func (m *MetricsServer) getLastHits() int64 {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	return m.lastHits
}

func (m *MetricsServer) getLastMisses() int64 {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	return m.lastMisses
}

func (m *MetricsServer) getLastEvictions() int64 {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	return m.lastEvictions
}

func (m *MetricsServer) getLastMetadataHits() int64 {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	return m.lastMetadataHits
}

func (m *MetricsServer) getLastMetadataMisses() int64 {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	return m.lastMetadataMisses
}

func (m *MetricsServer) storeLastValues(stats HelmCacheStats) {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	m.lastHits = stats.Hits
	m.lastMisses = stats.Misses
	m.lastEvictions = stats.Evictions
	m.lastMetadataHits = stats.MetadataHits
	m.lastMetadataMisses = stats.MetadataMisses
}
