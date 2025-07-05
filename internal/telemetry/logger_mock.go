package telemetry

import (
	"github.com/go-logr/logr"
)

// MockLoggerBackend is a mock implementation of LoggerBackendInterface for testing
type MockLoggerBackend struct {
	// CreateLoggerFunc can be set to customize the behavior of CreateLogger
	CreateLoggerFunc func(opts LoggerOptions) logr.Logger
	// CallCount tracks how many times CreateLogger was called
	CallCount int
	// LastOptions stores the last options passed to CreateLogger
	LastOptions LoggerOptions
}

// CreateLogger implements LoggerBackendInterface for testing
func (m *MockLoggerBackend) CreateLogger(opts LoggerOptions) logr.Logger {
	m.CallCount++
	m.LastOptions = opts

	if m.CreateLoggerFunc != nil {
		return m.CreateLoggerFunc(opts)
	}

	// Default to returning a discard logger for testing
	return logr.Discard()
}

// RegisterMockLoggerBackend registers the mock backend for testing
// This should only be used in test files
func RegisterMockLoggerBackend() *MockLoggerBackend {
	mock := &MockLoggerBackend{}
	RegisterLoggerBackend("mock", mock)
	return mock
}

// CacheStats represents cache statistics (needed for testing)
type CacheStats struct {
	Hits      int64
	Misses    int64
	Size      int64
	MaxSize   int64
	Evictions int64
	Keys      int
}

// MockCacheStatsProvider is a mock implementation of CacheStatsProvider for testing
type MockCacheStatsProvider struct {
	CacheStats CacheStats
}

// NewMockCacheStatsProvider creates a new mock cache stats provider
func NewMockCacheStatsProvider() *MockCacheStatsProvider {
	return &MockCacheStatsProvider{
		CacheStats: CacheStats{
			Hits:      10,
			Misses:    2,
			Size:      1024,
			MaxSize:   10240,
			Evictions: 1,
			Keys:      5,
		},
	}
}

// GetEntries returns the number of cache entries
func (m *MockCacheStatsProvider) GetEntries() int {
	return m.CacheStats.Keys
}

// GetCurrentSize returns the current cache size
func (m *MockCacheStatsProvider) GetCurrentSize() int64 {
	return m.CacheStats.Size
}

// GetMaxSize returns the maximum cache size
func (m *MockCacheStatsProvider) GetMaxSize() int64 {
	return m.CacheStats.MaxSize
}

// GetMaxEntries returns the maximum number of entries
func (m *MockCacheStatsProvider) GetMaxEntries() int {
	return 1000 // arbitrary default
}

// GetHits returns the number of cache hits
func (m *MockCacheStatsProvider) GetHits() int64 {
	return m.CacheStats.Hits
}

// GetMisses returns the number of cache misses
func (m *MockCacheStatsProvider) GetMisses() int64 {
	return m.CacheStats.Misses
}

// GetEvictions returns the number of evictions
func (m *MockCacheStatsProvider) GetEvictions() int64 {
	return m.CacheStats.Evictions
}

// GetHitRate returns the cache hit rate
func (m *MockCacheStatsProvider) GetHitRate() float64 {
	total := m.CacheStats.Hits + m.CacheStats.Misses
	if total == 0 {
		return 0
	}
	return float64(m.CacheStats.Hits) / float64(total) * 100
}

// GetUsagePercent returns the cache usage percentage
func (m *MockCacheStatsProvider) GetUsagePercent() float64 {
	if m.CacheStats.MaxSize == 0 {
		return 0
	}
	return float64(m.CacheStats.Size) / float64(m.CacheStats.MaxSize) * 100
}

// GetMetadataEntries returns the number of metadata entries
func (m *MockCacheStatsProvider) GetMetadataEntries() int {
	return 0
}

// GetMetadataHits returns the number of metadata hits
func (m *MockCacheStatsProvider) GetMetadataHits() int64 {
	return 0
}

// GetMetadataMisses returns the number of metadata misses
func (m *MockCacheStatsProvider) GetMetadataMisses() int64 {
	return 0
}

// GetMetadataHitRate returns the metadata hit rate
func (m *MockCacheStatsProvider) GetMetadataHitRate() float64 {
	return 0
}
