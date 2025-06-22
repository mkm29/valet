package helm

// Determine if a remote chart contains a values.schema.json file

import (
	"fmt"
	"os"
	"sync"

	"github.com/mkm29/valet/internal/config"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/getter"
)

const (
	RegistryTypeHTTP  = "HTTP"
	RegistryTypeHTTPS = "HTTPS"
	RegistryTypeOCI   = "OCI"

	// DefaultMaxChartSize is the default maximum size for charts to be cached (1MB, matching etcd limit)
	DefaultMaxChartSize = 1 * 1024 * 1024 // 1MB in bytes
)

// Helm provides functionality for working with Helm charts
type Helm struct {
	logger       *zap.Logger
	debug        bool
	cache        *chartCache
	maxChartSize int64
}

// HelmOptions configures a Helm instance
type HelmOptions struct {
	Debug        bool
	Logger       *zap.Logger
	MaxChartSize int64 // Maximum size in bytes for charts to be cached (0 = use default)
}

type chartCache struct {
	mu     sync.RWMutex
	charts map[string]*chart.Chart
}

// NewHelm creates a new Helm instance with options
func NewHelm(opts HelmOptions) *Helm {
	logger := opts.Logger
	if logger == nil {
		logger = zap.L().Named("helm")
	}

	maxSize := opts.MaxChartSize
	if maxSize <= 0 {
		maxSize = DefaultMaxChartSize
	}

	return &Helm{
		logger:       logger,
		debug:        opts.Debug,
		maxChartSize: maxSize,
		cache: &chartCache{
			charts: make(map[string]*chart.Chart),
		},
	}
}

// NewHelmWithDebug creates a new Helm instance with just debug flag (convenience function)
func NewHelmWithDebug(debug bool) *Helm {
	return NewHelm(HelmOptions{
		Debug: debug,
	})
}

// GetOptions builds getter options from a HelmChart configuration
func (h *Helm) GetOptions(c *config.HelmChart) []getter.Option {
	var getterOpts []getter.Option

	if c.Registry.Type == RegistryTypeHTTP {
		getterOpts = append(getterOpts, getter.WithPlainHTTP(true))
	}
	if c.Registry.Auth != nil && c.Registry.Auth.Username != "" && c.Registry.Auth.Password != "" {
		getterOpts = append(getterOpts, getter.WithBasicAuth(c.Registry.Auth.Username, c.Registry.Auth.Password))
	}
	if c.Registry.Insecure {
		getterOpts = append(getterOpts, getter.WithInsecureSkipVerifyTLS(true))
	}
	if c.Registry.TLS != nil && c.Registry.TLS.CertFile != "" && c.Registry.TLS.KeyFile != "" && c.Registry.TLS.CaFile != "" {
		getterOpts = append(getterOpts, getter.WithTLSClientConfig(c.Registry.TLS.CertFile, c.Registry.TLS.KeyFile, c.Registry.TLS.CaFile))
		getterOpts = append(getterOpts, getter.WithURL(c.Registry.URL))
	}

	return getterOpts
}

func (h *Helm) getOrLoadChart(c *config.HelmChart) (*chart.Chart, error) {
	// Create a cache key from chart name and version
	cacheKey := fmt.Sprintf("%s/%s@%s", c.Registry.URL, c.Name, c.Version)

	// Check if chart is already in cache (read lock)
	h.cache.mu.RLock()
	if cachedChart, ok := h.cache.charts[cacheKey]; ok {
		h.cache.mu.RUnlock()
		if h.debug {
			h.logger.Debug("Chart found in cache",
				zap.String("name", c.Name),
				zap.String("version", c.Version),
				zap.String("cacheKey", cacheKey),
			)
		}
		return cachedChart, nil
	}
	h.cache.mu.RUnlock()

	// Chart not in cache, need to load it (write lock)
	h.cache.mu.Lock()
	defer h.cache.mu.Unlock()

	// Double-check after acquiring write lock
	if cachedChart, ok := h.cache.charts[cacheKey]; ok {
		if h.debug {
			h.logger.Debug("Chart found in cache after acquiring write lock",
				zap.String("name", c.Name),
				zap.String("version", c.Version),
				zap.String("cacheKey", cacheKey),
			)
		}
		return cachedChart, nil
	}

	// Load the chart
	if h.debug {
		h.logger.Debug("Loading chart from registry",
			zap.String("name", c.Name),
			zap.String("version", c.Version),
			zap.String("cacheKey", cacheKey),
		)
	}

	loadedChart, err := h.loadChart(c)
	if err != nil {
		return nil, err
	}

	// Store in cache
	h.cache.charts[cacheKey] = loadedChart

	// Calculate total cache size
	var totalCacheSize int64
	for _, cachedChart := range h.cache.charts {
		for _, file := range cachedChart.Raw {
			totalCacheSize += int64(len(file.Data))
		}
	}

	if h.debug {
		h.logger.Debug("Chart cached successfully",
			zap.String("name", c.Name),
			zap.String("version", c.Version),
			zap.String("cacheKey", cacheKey),
			zap.Int("cacheCount", len(h.cache.charts)),
			zap.String("totalCacheSize", h.formatBytes(totalCacheSize)),
		)
	}

	return loadedChart, nil
}

// loadChart downloads and loads a Helm chart from the specified registry
func (h *Helm) loadChart(c *config.HelmChart) (*chart.Chart, error) {
	url := fmt.Sprintf("%s/%s-%s.tgz", c.Registry.URL, c.Name, c.Version)

	if h.debug {
		h.logger.Debug("Loading chart",
			zap.String("name", c.Name),
			zap.String("version", c.Version),
			zap.String("url", url),
			zap.Int64("maxSizeBytes", h.maxChartSize),
		)
	}

	// Create appropriate getter based on registry type
	var g getter.Getter
	var err error

	switch c.Registry.Type {
	case RegistryTypeHTTP, RegistryTypeHTTPS:
		g, err = getter.NewHTTPGetter()
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP getter: %w", err)
		}
	case RegistryTypeOCI:
		g, err = getter.NewOCIGetter()
		if err != nil {
			return nil, fmt.Errorf("failed to create OCI getter: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported registry type: %s", c.Registry.Type)
	}

	// Get the chart using configured options
	getterOpts := h.GetOptions(c)
	provider, err := g.Get(url, getterOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to get chart: %w", err)
	}

	// Check the size before loading
	chartSize := int64(provider.Len())
	if h.debug {
		h.logger.Debug("Chart file size",
			zap.String("name", c.Name),
			zap.String("version", c.Version),
			zap.Int64("sizeBytes", chartSize),
			zap.String("sizeHuman", h.formatBytes(chartSize)),
		)
	}

	// Check if chart exceeds size limit
	if chartSize > h.maxChartSize {
		return nil, fmt.Errorf("chart size (%s) exceeds maximum allowed size (%s)",
			h.formatBytes(chartSize), h.formatBytes(h.maxChartSize))
	}

	// Load the chart archive
	chart, err := loader.LoadArchive(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart: %w", err)
	}

	if h.debug {
		h.logger.Debug("Chart loaded successfully",
			zap.String("name", chart.Name()),
			zap.String("version", chart.Metadata.Version),
			zap.Int64("sizeBytes", chartSize),
		)
	}

	return chart, nil
}

// getSchemaFile retrieves the values.schema.json file from the chart if it exists.
// This method is the single source of truth for finding schema files in charts,
// eliminating duplication between HasSchema and DownloadSchema methods.
// Returns nil if the schema file doesn't exist (not an error condition).
func (h *Helm) getSchemaFile(c *config.HelmChart) (*chart.File, error) {
	// Load the chart using caching logic
	chart, err := h.getOrLoadChart(c)
	if err != nil {
		return nil, fmt.Errorf("error loading chart: %w", err)
	}

	// Find the values.schema.json file
	for _, file := range chart.Raw {
		if h.debug {
			h.logger.Debug("Checking file", zap.String("file", file.Name))
		}
		if file.Name == "values.schema.json" {
			if h.debug {
				h.logger.Debug("Found values.schema.json in chart")
			}
			return file, nil
		}
	}

	if h.debug {
		h.logger.Debug("Chart does not have values.schema.json")
	}
	return nil, nil
}

// HasSchema checks if a chart has a values.schema.json file
func (h *Helm) HasSchema(c *config.HelmChart) (bool, error) {
	file, err := h.getSchemaFile(c)
	if err != nil {
		return false, err
	}
	return file != nil, nil
}

// GetSchemaBytes retrieves the values.schema.json file content as bytes
func (h *Helm) GetSchemaBytes(c *config.HelmChart) ([]byte, error) {
	file, err := h.getSchemaFile(c)
	if err != nil {
		return nil, err
	}

	if file == nil {
		return nil, fmt.Errorf("values.schema.json not found in chart")
	}

	return file.Data, nil
}

// DownloadSchema retrieves the values.schema.json file from the chart and saves to temporary file
func (h *Helm) DownloadSchema(c *config.HelmChart) (string, error) {
	file, err := h.getSchemaFile(c)
	if err != nil {
		return "", err
	}

	if file == nil {
		return "", fmt.Errorf("values.schema.json not found in chart")
	}

	// Write the schema to a temporary file
	tmp, err := os.CreateTemp("", "values.schema.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer tmp.Close()
	// change permissions on the temporary file
	if err := os.Chmod(tmp.Name(), 0600); err != nil {
		return "", fmt.Errorf("failed to set permissions on temporary file: %w", err)
	}

	if _, err := tmp.Write(file.Data); err != nil {
		return "", fmt.Errorf("failed to write to temporary file: %w", err)
	}

	if h.debug {
		h.logger.Debug("Schema saved to temporary file", zap.String("path", tmp.Name()))
	}

	return tmp.Name(), nil
}

// formatBytes converts bytes to human-readable format
func (h *Helm) formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
