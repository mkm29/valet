package helm

// Determine if a remote chart contains a values.schema.json file

import (
	"container/list"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/mkm29/valet/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/getter"
)

const (
	RegistryTypeHTTP  = "HTTP"
	RegistryTypeHTTPS = "HTTPS"
	RegistryTypeOCI   = "OCI"

	// DefaultMaxChartSize is the default maximum size for individual charts (1MB, matching etcd limit)
	DefaultMaxChartSize = 1 * 1024 * 1024 // 1MB in bytes

	// DefaultMaxCacheSize is the default maximum total size for the cache (10MB)
	DefaultMaxCacheSize = 10 * 1024 * 1024 // 10MB in bytes

	// DefaultMaxCacheEntries is the default maximum number of charts in cache
	DefaultMaxCacheEntries = 50
)

// Helm provides functionality for working with Helm charts
type Helm struct {
	logger          *zap.Logger
	debug           bool
	cache           *chartCache
	maxChartSize    int64
	maxCacheSize    int64
	maxCacheEntries int
}

// HelmOptions configures a Helm instance
type HelmOptions struct {
	Debug           bool
	Logger          *zap.Logger
	MaxChartSize    int64 // Maximum size in bytes for individual charts (0 = use default)
	MaxCacheSize    int64 // Maximum total size in bytes for cache (0 = use default)
	MaxCacheEntries int   // Maximum number of entries in cache (0 = use default)
}

// cacheEntry represents a cached chart with metadata
type cacheEntry struct {
	chart      *chart.Chart
	size       int64
	lastAccess time.Time
	key        string
}

// metadataEntry represents cached chart metadata for faster schema checks
type metadataEntry struct {
	hasSchema    bool
	chartName    string
	chartVersion string
	lastAccess   time.Time
}

type chartCache struct {
	mu           sync.RWMutex
	entries      map[string]*cacheEntry
	lruList      *list.List               // LRU list of cache keys
	keyToElement map[string]*list.Element // Map from key to list element
	currentSize  int64                    // Current total size of cached charts
	hits         int64                    // Cache hit count
	misses       int64                    // Cache miss count
	evictions    int64                    // Number of evictions

	// Metadata cache for faster schema checks
	metadataMu      sync.RWMutex
	metadata        map[string]*metadataEntry
	metadataLRU     *list.List
	metadataKeyToEl map[string]*list.Element
	metadataHits    int64
	metadataMisses  int64
}

// NewHelm creates a new Helm instance with options
func NewHelm(opts HelmOptions) *Helm {
	logger := opts.Logger
	if logger == nil {
		logger = zap.L().Named("helm")
	}

	maxChartSize := opts.MaxChartSize
	if maxChartSize <= 0 {
		maxChartSize = DefaultMaxChartSize
	}

	maxCacheSize := opts.MaxCacheSize
	if maxCacheSize <= 0 {
		maxCacheSize = DefaultMaxCacheSize
	}

	maxCacheEntries := opts.MaxCacheEntries
	if maxCacheEntries <= 0 {
		maxCacheEntries = DefaultMaxCacheEntries
	}

	return &Helm{
		logger:          logger,
		debug:           opts.Debug,
		maxChartSize:    maxChartSize,
		maxCacheSize:    maxCacheSize,
		maxCacheEntries: maxCacheEntries,
		cache: &chartCache{
			entries:      make(map[string]*cacheEntry),
			lruList:      list.New(),
			keyToElement: make(map[string]*list.Element),
			currentSize:  0,
			hits:         0,
			misses:       0,
			evictions:    0,
			// Initialize metadata cache
			metadata:        make(map[string]*metadataEntry),
			metadataLRU:     list.New(),
			metadataKeyToEl: make(map[string]*list.Element),
			metadataHits:    0,
			metadataMisses:  0,
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

// calculateChartSize calculates the total size of a chart by summing all file sizes.
// This includes all files in the chart's Raw field (Chart.yaml, values.yaml, templates, etc.)
func (h *Helm) calculateChartSize(ch *chart.Chart) int64 {
	var size int64
	for _, file := range ch.Raw {
		size += int64(len(file.Data))
	}
	return size
}

// evictLRU evicts least recently used entries until we're under both size and count limits.
// This method implements the LRU eviction policy to prevent unbounded cache growth.
//
// The eviction continues until BOTH conditions are satisfied:
// 1. Total cache size is under h.maxCacheSize
// 2. Number of entries is less than h.maxCacheEntries
//
// IMPORTANT: Must be called with write lock held on h.cache.mu
func (h *Helm) evictLRU() {
	// Continue evicting while we exceed either limit
	for h.cache.currentSize > h.maxCacheSize || len(h.cache.entries) >= h.maxCacheEntries {
		// Get the least recently used element from the back of the list
		elem := h.cache.lruList.Back()
		if elem == nil {
			// Safety check: list is empty, nothing to evict
			break
		}

		// Extract the cache key and find the corresponding entry
		key := elem.Value.(string)
		entry := h.cache.entries[key]

		// Remove from all cache data structures:
		// 1. Remove from entries map
		delete(h.cache.entries, key)
		// 2. Remove from key-to-element mapping
		delete(h.cache.keyToElement, key)
		// 3. Remove from LRU list
		h.cache.lruList.Remove(elem)

		// Update cache statistics
		h.cache.currentSize -= entry.size
		h.cache.evictions++

		if h.debug {
			h.logger.Debug("Evicted chart from cache",
				zap.String("key", key),
				zap.String("size", h.formatBytes(entry.size)),
				zap.Int64("evictions", h.cache.evictions),
				zap.String("reason", h.getEvictionReason()),
			)
		}
	}
}

// getEvictionReason returns a string describing why eviction is happening
func (h *Helm) getEvictionReason() string {
	sizeExceeded := h.cache.currentSize > h.maxCacheSize
	countExceeded := len(h.cache.entries) >= h.maxCacheEntries

	if sizeExceeded && countExceeded {
		return "both size and count limits exceeded"
	} else if sizeExceeded {
		return "size limit exceeded"
	} else if countExceeded {
		return "count limit exceeded"
	}
	return "unknown"
}

// updateLRU moves an existing entry to the front of the LRU list or adds a new entry.
// The front of the list contains the most recently used items.
//
// IMPORTANT: Must be called with write lock held on h.cache.mu
func (h *Helm) updateLRU(key string) {
	if elem, exists := h.cache.keyToElement[key]; exists {
		// Entry exists: move it to the front (mark as recently used)
		h.cache.lruList.MoveToFront(elem)
	} else {
		// New entry: add to the front of the list
		elem := h.cache.lruList.PushFront(key)
		// Store the list element reference for O(1) access later
		h.cache.keyToElement[key] = elem
	}
}

// updateMetadataLRU updates the metadata LRU list when an entry is accessed.
// IMPORTANT: Must be called with write lock held on h.cache.metadataMu
func (h *Helm) updateMetadataLRU(key string) {
	if elem, exists := h.cache.metadataKeyToEl[key]; exists {
		// Entry exists: move it to the front (mark as recently used)
		h.cache.metadataLRU.MoveToFront(elem)
	} else {
		// New entry: add to the front of the list
		elem := h.cache.metadataLRU.PushFront(key)
		// Store the list element reference for O(1) access later
		h.cache.metadataKeyToEl[key] = elem
	}
}

// evictMetadataLRU evicts least recently used metadata entries
// IMPORTANT: Must be called with write lock held on h.cache.metadataMu
func (h *Helm) evictMetadataLRU() {
	// Keep metadata cache at 2x the size of chart cache for better hit rate
	maxMetadataEntries := h.maxCacheEntries * 2

	for len(h.cache.metadata) >= maxMetadataEntries {
		elem := h.cache.metadataLRU.Back()
		if elem == nil {
			break
		}

		key := elem.Value.(string)
		delete(h.cache.metadata, key)
		delete(h.cache.metadataKeyToEl, key)
		h.cache.metadataLRU.Remove(elem)

		if h.debug {
			h.logger.Debug("Evicted metadata from cache",
				zap.String("key", key),
				zap.Int("remainingEntries", len(h.cache.metadata)),
			)
		}
	}
}

// updateMetadataCache updates the metadata cache with chart information
func (h *Helm) updateMetadataCache(cacheKey string, ch *chart.Chart, c *config.HelmChart) {
	// Check if chart has schema
	hasSchema := false
	for _, file := range ch.Raw {
		if file.Name == "values.schema.json" {
			hasSchema = true
			break
		}
	}

	// Update metadata cache
	h.cache.metadataMu.Lock()
	defer h.cache.metadataMu.Unlock()

	// Evict if necessary
	maxMetadataEntries := h.maxCacheEntries * 2
	if len(h.cache.metadata) >= maxMetadataEntries {
		h.evictMetadataLRU()
	}

	// Add to metadata cache
	h.cache.metadata[cacheKey] = &metadataEntry{
		hasSchema:    hasSchema,
		chartName:    c.Name,
		chartVersion: c.Version,
		lastAccess:   time.Now(),
	}
	h.updateMetadataLRU(cacheKey)

	if h.debug {
		h.logger.Debug("Updated metadata cache",
			zap.String("chart", fmt.Sprintf("%s/%s", c.Name, c.Version)),
			zap.Bool("hasSchema", hasSchema),
			zap.Int("metadataEntries", len(h.cache.metadata)),
		)
	}
}

// getOrLoadChart retrieves a chart from cache or loads it from the registry.
// This is the main entry point for all chart operations and implements:
// 1. Thread-safe cache lookup with read/write lock optimization
// 2. LRU cache management with eviction
// 3. Size-based cache limits
// 4. Comprehensive hit/miss tracking
func (h *Helm) getOrLoadChart(c *config.HelmChart) (*chart.Chart, error) {
	// Create a unique cache key combining registry URL, chart name, and version
	// Format: "https://charts.example.com/repo/mychart@1.2.3"
	cacheKey := fmt.Sprintf("%s/%s@%s", c.Registry.URL, c.Name, c.Version)

	// PHASE 1: Optimistic read with read lock (most common case - cache hit)
	h.cache.mu.RLock()
	if entry, ok := h.cache.entries[cacheKey]; ok {
		// Found in cache - release read lock before acquiring write lock
		h.cache.mu.RUnlock()

		// PHASE 2: Update cache metadata (requires write lock)
		h.cache.mu.Lock()
		entry.lastAccess = time.Now()
		h.updateLRU(cacheKey) // Move to front of LRU list
		h.cache.hits++

		// Calculate current hit rate for logging
		totalRequests := h.cache.hits + h.cache.misses
		hitRate := float64(0)
		if totalRequests > 0 {
			hitRate = float64(h.cache.hits) / float64(totalRequests) * 100
		}
		h.cache.mu.Unlock()

		if h.debug {
			h.logger.Debug("Cache hit",
				zap.String("chart", fmt.Sprintf("%s/%s", c.Name, c.Version)),
				zap.String("cacheKey", cacheKey),
				zap.Int64("totalHits", h.cache.hits),
				zap.Int64("totalMisses", h.cache.misses),
				zap.Float64("hitRate", hitRate),
				zap.String("entrySize", h.formatBytes(entry.size)),
				zap.Duration("age", time.Since(entry.lastAccess)),
			)
		}
		return entry.chart, nil
	}
	h.cache.mu.RUnlock()

	// PHASE 3: Cache miss - need exclusive access for loading
	h.cache.mu.Lock()
	defer h.cache.mu.Unlock()

	// Double-check pattern: another goroutine might have loaded it while we waited for write lock
	if entry, ok := h.cache.entries[cacheKey]; ok {
		entry.lastAccess = time.Now()
		h.updateLRU(cacheKey)
		h.cache.hits++

		if h.debug {
			h.logger.Debug("Cache hit (after write lock)",
				zap.String("chart", fmt.Sprintf("%s/%s", c.Name, c.Version)),
				zap.String("cacheKey", cacheKey),
			)
		}
		return entry.chart, nil
	}

	// PHASE 4: Confirmed cache miss - update statistics
	h.cache.misses++
	missRate := float64(h.cache.misses) / float64(h.cache.hits+h.cache.misses) * 100

	if h.debug {
		h.logger.Debug("Cache miss - loading from registry",
			zap.String("chart", fmt.Sprintf("%s/%s", c.Name, c.Version)),
			zap.String("cacheKey", cacheKey),
			zap.Int64("totalMisses", h.cache.misses),
			zap.Float64("missRate", missRate),
			zap.String("registry", c.Registry.URL),
		)
	}

	// PHASE 5: Load chart from registry
	startTime := time.Now()
	loadedChart, err := h.loadChart(c)
	if err != nil {
		// Don't cache failures
		return nil, err
	}
	loadDuration := time.Since(startTime)

	// PHASE 6: Evaluate cacheability
	chartSize := h.calculateChartSize(loadedChart)

	// Check if chart is too large for our cache
	if chartSize > h.maxCacheSize {
		if h.debug || h.logger.Core().Enabled(zapcore.WarnLevel) {
			h.logger.Warn("Chart too large to cache",
				zap.String("chart", fmt.Sprintf("%s/%s", c.Name, c.Version)),
				zap.String("chartSize", h.formatBytes(chartSize)),
				zap.String("maxCacheSize", h.formatBytes(h.maxCacheSize)),
				zap.Float64("sizeRatio", float64(chartSize)/float64(h.maxCacheSize)),
			)
		}
		// Return the chart without caching
		return loadedChart, nil
	}

	// PHASE 7: Make room in cache if necessary
	evictionStart := time.Now()
	evictionCount := 0
	for h.cache.currentSize+chartSize > h.maxCacheSize || len(h.cache.entries) >= h.maxCacheEntries {
		h.evictLRU()
		evictionCount++
	}
	evictionDuration := time.Since(evictionStart)

	// PHASE 8: Add to cache
	entry := &cacheEntry{
		chart:      loadedChart,
		size:       chartSize,
		lastAccess: time.Now(),
		key:        cacheKey,
	}

	h.cache.entries[cacheKey] = entry
	h.cache.currentSize += chartSize
	h.updateLRU(cacheKey)

	// Also update metadata cache
	h.updateMetadataCache(cacheKey, loadedChart, c)

	// Calculate final cache state
	cacheUsagePercent := float64(h.cache.currentSize) / float64(h.maxCacheSize) * 100
	entriesUsagePercent := float64(len(h.cache.entries)) / float64(h.maxCacheEntries) * 100

	if h.debug {
		h.logger.Debug("Chart cached successfully",
			zap.String("chart", fmt.Sprintf("%s/%s", c.Name, c.Version)),
			zap.String("cacheKey", cacheKey),
			zap.Duration("loadTime", loadDuration),
			zap.String("chartSize", h.formatBytes(chartSize)),
			zap.Int("evictedEntries", evictionCount),
			zap.Duration("evictionTime", evictionDuration),
			zap.Int("totalEntries", len(h.cache.entries)),
			zap.String("cacheSize", h.formatBytes(h.cache.currentSize)),
			zap.Float64("cacheSizeUsage", cacheUsagePercent),
			zap.Float64("cacheEntriesUsage", entriesUsagePercent),
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
		// Provide detailed error message for troubleshooting
		errMsg := fmt.Sprintf("failed to download chart from %s: %v", url, err)

		// Add context-specific troubleshooting hints
		if c.Registry.Type == RegistryTypeHTTP || c.Registry.Type == RegistryTypeHTTPS {
			errMsg += fmt.Sprintf("\n\nTroubleshooting hints for %s registry:", c.Registry.Type)
			errMsg += fmt.Sprintf("\n- Verify the registry URL is correct: %s", c.Registry.URL)
			errMsg += fmt.Sprintf("\n- Check if the chart exists: %s/%s", c.Name, c.Version)
			errMsg += "\n- Ensure the registry is accessible from your network"

			if c.Registry.Auth != nil && (c.Registry.Auth.Username != "" || c.Registry.Auth.Token != "") {
				errMsg += "\n- Verify your authentication credentials are correct"
			}

			if c.Registry.Type == RegistryTypeHTTPS && c.Registry.Insecure {
				errMsg += "\n- You're using insecure HTTPS, ensure the registry supports this"
			}

			if c.Registry.TLS != nil && c.Registry.TLS.InsecureSkipTLSVerify {
				errMsg += "\n- TLS verification is disabled, this may cause security warnings"
			}
		} else if c.Registry.Type == RegistryTypeOCI {
			errMsg += "\n\nTroubleshooting hints for OCI registry:"
			errMsg += fmt.Sprintf("\n- Verify the OCI registry URL format: %s", c.Registry.URL)
			errMsg += fmt.Sprintf("\n- Expected format: oci://registry.example.com/namespace/%s", c.Name)
			errMsg += "\n- Ensure you have proper authentication for OCI registries"
			errMsg += "\n- Check if the OCI registry requires specific authentication methods"
		}

		errMsg += "\n\nCommon issues:"
		errMsg += "\n- Network connectivity problems"
		errMsg += "\n- Incorrect chart name or version"
		errMsg += "\n- Missing or incorrect authentication"
		errMsg += "\n- Registry URL format issues"

		return nil, fmt.Errorf("%s", errMsg)
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
		errMsg := fmt.Sprintf("failed to load chart archive for %s/%s: %v", c.Name, c.Version, err)
		errMsg += "\n\nPossible causes:"
		errMsg += "\n- The downloaded file is not a valid Helm chart archive"
		errMsg += "\n- The chart archive is corrupted or incomplete"
		errMsg += "\n- The chart format is incompatible with this version of Helm"
		errMsg += fmt.Sprintf("\n- Downloaded size: %s", h.formatBytes(chartSize))

		return nil, fmt.Errorf("%s", errMsg)
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
		return nil, fmt.Errorf("error loading chart %s/%s: %w", c.Name, c.Version, err)
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
// This method first checks the metadata cache for fast lookups
func (h *Helm) HasSchema(c *config.HelmChart) (bool, error) {
	// Create cache key
	cacheKey := fmt.Sprintf("%s/%s@%s", c.Registry.URL, c.Name, c.Version)

	// First check metadata cache for fast lookup
	h.cache.metadataMu.RLock()
	if entry, ok := h.cache.metadata[cacheKey]; ok {
		// Found in metadata cache
		h.cache.metadataMu.RUnlock()

		// Update access time and stats
		h.cache.metadataMu.Lock()
		entry.lastAccess = time.Now()
		h.updateMetadataLRU(cacheKey)
		h.cache.metadataHits++

		totalRequests := h.cache.metadataHits + h.cache.metadataMisses
		hitRate := float64(0)
		if totalRequests > 0 {
			hitRate = float64(h.cache.metadataHits) / float64(totalRequests) * 100
		}
		h.cache.metadataMu.Unlock()

		if h.debug {
			h.logger.Debug("Metadata cache hit",
				zap.String("chart", fmt.Sprintf("%s/%s", c.Name, c.Version)),
				zap.Bool("hasSchema", entry.hasSchema),
				zap.Int64("totalHits", h.cache.metadataHits),
				zap.Float64("hitRate", hitRate),
			)
		}

		return entry.hasSchema, nil
	}
	h.cache.metadataMu.RUnlock()

	// Metadata cache miss - update stats
	h.cache.metadataMu.Lock()
	h.cache.metadataMisses++
	totalRequests := h.cache.metadataHits + h.cache.metadataMisses
	missRate := float64(0)
	if totalRequests > 0 {
		missRate = float64(h.cache.metadataMisses) / float64(totalRequests) * 100
	}
	h.cache.metadataMu.Unlock()

	if h.debug {
		h.logger.Debug("Metadata cache miss",
			zap.String("chart", fmt.Sprintf("%s/%s", c.Name, c.Version)),
			zap.Int64("totalMisses", h.cache.metadataMisses),
			zap.Float64("missRate", missRate),
		)
	}

	// Fall back to loading the chart
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
func (h *Helm) DownloadSchema(c *config.HelmChart) (string, func(), error) {
	file, err := h.getSchemaFile(c)

	if err != nil {
		return "", nil, err
	}

	if file == nil {
		return "", nil, fmt.Errorf("values.schema.json not found in chart")
	}

	// Write the schema to a temporary file
	tmp, err := os.CreateTemp("", "values.schema.json")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer tmp.Close()
	// change permissions on the temporary file
	if err := os.Chmod(tmp.Name(), 0600); err != nil {
		return "", nil, fmt.Errorf("failed to set permissions on temporary file: %w", err)
	}

	if _, err := tmp.Write(file.Data); err != nil {
		return "", nil, fmt.Errorf("failed to write to temporary file: %w", err)
	}

	if h.debug {
		h.logger.Debug("Schema saved to temporary file", zap.String("path", tmp.Name()))
	}
	cleanup := func() { os.Remove(tmp.Name()) }

	return tmp.Name(), cleanup, nil
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

// CacheStats represents cache statistics
type CacheStats struct {
	Entries      int     `json:"entries"`
	CurrentSize  int64   `json:"currentSize"`
	MaxSize      int64   `json:"maxSize"`
	MaxEntries   int     `json:"maxEntries"`
	Hits         int64   `json:"hits"`
	Misses       int64   `json:"misses"`
	Evictions    int64   `json:"evictions"`
	HitRate      float64 `json:"hitRate"`
	UsagePercent float64 `json:"usagePercent"`

	// Metadata cache stats
	MetadataEntries int     `json:"metadataEntries"`
	MetadataHits    int64   `json:"metadataHits"`
	MetadataMisses  int64   `json:"metadataMisses"`
	MetadataHitRate float64 `json:"metadataHitRate"`
}

// GetCacheStats returns current cache statistics
func (h *Helm) GetCacheStats() CacheStats {
	h.cache.mu.RLock()
	defer h.cache.mu.RUnlock()

	totalRequests := h.cache.hits + h.cache.misses
	hitRate := float64(0)
	if totalRequests > 0 {
		hitRate = float64(h.cache.hits) / float64(totalRequests) * 100
	}

	usagePercent := float64(0)
	if h.maxCacheSize > 0 {
		usagePercent = float64(h.cache.currentSize) / float64(h.maxCacheSize) * 100
	}

	// Get metadata cache stats
	h.cache.metadataMu.RLock()
	metadataEntries := len(h.cache.metadata)
	metadataHits := h.cache.metadataHits
	metadataMisses := h.cache.metadataMisses
	h.cache.metadataMu.RUnlock()

	metadataTotalRequests := metadataHits + metadataMisses
	metadataHitRate := float64(0)
	if metadataTotalRequests > 0 {
		metadataHitRate = float64(metadataHits) / float64(metadataTotalRequests) * 100
	}

	return CacheStats{
		Entries:         len(h.cache.entries),
		CurrentSize:     h.cache.currentSize,
		MaxSize:         h.maxCacheSize,
		MaxEntries:      h.maxCacheEntries,
		Hits:            h.cache.hits,
		Misses:          h.cache.misses,
		Evictions:       h.cache.evictions,
		HitRate:         hitRate,
		UsagePercent:    usagePercent,
		MetadataEntries: metadataEntries,
		MetadataHits:    metadataHits,
		MetadataMisses:  metadataMisses,
		MetadataHitRate: metadataHitRate,
	}
}

// ClearCache clears all entries from the cache
func (h *Helm) ClearCache() {
	h.cache.mu.Lock()
	defer h.cache.mu.Unlock()

	h.cache.entries = make(map[string]*cacheEntry)
	h.cache.lruList = list.New()
	h.cache.keyToElement = make(map[string]*list.Element)
	h.cache.currentSize = 0
	// Don't reset statistics, they're cumulative

	// Also clear metadata cache
	h.cache.metadataMu.Lock()
	h.cache.metadata = make(map[string]*metadataEntry)
	h.cache.metadataLRU = list.New()
	h.cache.metadataKeyToEl = make(map[string]*list.Element)
	h.cache.metadataMu.Unlock()

	if h.debug {
		h.logger.Debug("Cache cleared (including metadata)")
	}
}
