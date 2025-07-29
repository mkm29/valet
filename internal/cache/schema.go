package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SchemaCache provides caching for generated schemas based on file modification times
type SchemaCache struct {
	entries map[string]*CacheEntry
	mu      sync.RWMutex
	// In-memory memoization for inferSchema calls
	memoCache map[string]map[string]any
	memoMu    sync.RWMutex
}

// CacheEntry stores cached schema data
type CacheEntry struct {
	Schema   map[string]any `json:"schema"`
	ModTime  time.Time      `json:"mod_time"`
	Checksum string         `json:"checksum"`
}

// NewSchemaCache creates a new schema cache instance
func NewSchemaCache() *SchemaCache {
	return &SchemaCache{
		entries:   make(map[string]*CacheEntry),
		memoCache: make(map[string]map[string]any),
	}
}

// GetOrGenerate retrieves cached schema or generates new one if cache miss
func (sc *SchemaCache) GetOrGenerate(path string, generator func() (map[string]any, error)) (map[string]any, error) {
	// Get file info
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// Calculate file checksum for additional validation
	checksum, err := calculateFileChecksum(path)
	if err != nil {
		return nil, err
	}

	// Check cache
	sc.mu.RLock()
	entry, ok := sc.entries[path]
	sc.mu.RUnlock()

	// Return cached entry if valid
	if ok && entry.ModTime.Equal(info.ModTime()) && entry.Checksum == checksum {
		return entry.Schema, nil
	}

	// Generate new schema
	schema, err := generator()
	if err != nil {
		return nil, err
	}

	// Update cache
	sc.mu.Lock()
	sc.entries[path] = &CacheEntry{
		Schema:   schema,
		ModTime:  info.ModTime(),
		Checksum: checksum,
	}
	sc.mu.Unlock()

	return schema, nil
}

// MemoizeInferSchema provides memoization for schema inference
func (sc *SchemaCache) MemoizeInferSchema(key string, generator func() map[string]any) map[string]any {
	// Check cache first
	sc.memoMu.RLock()
	if cached, ok := sc.memoCache[key]; ok {
		sc.memoMu.RUnlock()
		return cached
	}
	sc.memoMu.RUnlock()

	// Generate and cache
	result := generator()
	
	sc.memoMu.Lock()
	sc.memoCache[key] = result
	sc.memoMu.Unlock()

	return result
}

// ClearMemoCache clears the in-memory memoization cache
func (sc *SchemaCache) ClearMemoCache() {
	sc.memoMu.Lock()
	sc.memoCache = make(map[string]map[string]any)
	sc.memoMu.Unlock()
}

// SaveToFile persists cache to disk
func (sc *SchemaCache) SaveToFile(cachePath string) error {
	sc.mu.RLock()
	data, err := json.MarshalIndent(sc.entries, "", "  ")
	sc.mu.RUnlock()

	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(cachePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Write cache file
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// LoadFromFile loads cache from disk
func (sc *SchemaCache) LoadFromFile(cachePath string) error {
	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Cache file doesn't exist, that's ok
			return nil
		}
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	var entries map[string]*CacheEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("failed to unmarshal cache: %w", err)
	}

	sc.mu.Lock()
	sc.entries = entries
	sc.mu.Unlock()

	return nil
}

// calculateFileChecksum computes SHA256 checksum of a file
func calculateFileChecksum(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// GenerateMemoKey creates a unique key for memoization based on value and default value
func GenerateMemoKey(val, defaultVal any) string {
	// Create a deterministic string representation
	valStr := fmt.Sprintf("%T:%v", val, val)
	defStr := fmt.Sprintf("%T:%v", defaultVal, defaultVal)
	
	combined := valStr + "|" + defStr
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}