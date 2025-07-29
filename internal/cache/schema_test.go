package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSchemaCache(t *testing.T) {
	cache := NewSchemaCache()
	assert.NotNil(t, cache)
	assert.NotNil(t, cache.entries)
	assert.NotNil(t, cache.memoCache)
	assert.Empty(t, cache.entries)
	assert.Empty(t, cache.memoCache)
}

func TestSchemaCache_GetOrGenerate(t *testing.T) {
	cache := NewSchemaCache()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.yaml")

	// Create a test file
	content := []byte("test: value")
	err := os.WriteFile(testFile, content, 0644)
	require.NoError(t, err)

	// Test schema generation
	generatorCalled := 0
	expectedSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"test": map[string]any{
				"type":    "string",
				"default": "value",
			},
		},
	}

	generator := func() (map[string]any, error) {
		generatorCalled++
		return expectedSchema, nil
	}

	// First call - should generate
	schema1, err := cache.GetOrGenerate(testFile, generator)
	assert.NoError(t, err)
	assert.Equal(t, expectedSchema, schema1)
	assert.Equal(t, 1, generatorCalled)

	// Second call - should use cache
	schema2, err := cache.GetOrGenerate(testFile, generator)
	assert.NoError(t, err)
	assert.Equal(t, expectedSchema, schema2)
	assert.Equal(t, 1, generatorCalled) // Generator not called again

	// Modify file
	time.Sleep(10 * time.Millisecond)
	err = os.WriteFile(testFile, []byte("test: newvalue"), 0644)
	require.NoError(t, err)

	// Third call - should regenerate due to file modification
	schema3, err := cache.GetOrGenerate(testFile, generator)
	assert.NoError(t, err)
	assert.Equal(t, expectedSchema, schema3)
	assert.Equal(t, 2, generatorCalled)
}

func TestSchemaCache_GetOrGenerate_FileNotExist(t *testing.T) {
	cache := NewSchemaCache()
	nonExistentFile := "/tmp/non-existent-file.yaml"

	generator := func() (map[string]any, error) {
		return nil, nil
	}

	_, err := cache.GetOrGenerate(nonExistentFile, generator)
	assert.Error(t, err)
}

func TestSchemaCache_GetOrGenerate_GeneratorError(t *testing.T) {
	cache := NewSchemaCache()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.yaml")

	// Create a test file
	err := os.WriteFile(testFile, []byte("test: value"), 0644)
	require.NoError(t, err)

	expectedError := assert.AnError
	generator := func() (map[string]any, error) {
		return nil, expectedError
	}

	_, err = cache.GetOrGenerate(testFile, generator)
	assert.Equal(t, expectedError, err)
}

func TestSchemaCache_MemoizeInferSchema(t *testing.T) {
	cache := NewSchemaCache()

	callCount := 0
	expectedResult := map[string]any{"type": "string"}

	generator := func() map[string]any {
		callCount++
		return expectedResult
	}

	// First call
	result1 := cache.MemoizeInferSchema("key1", generator)
	assert.Equal(t, expectedResult, result1)
	assert.Equal(t, 1, callCount)

	// Second call with same key - should use memoized result
	result2 := cache.MemoizeInferSchema("key1", generator)
	assert.Equal(t, expectedResult, result2)
	assert.Equal(t, 1, callCount) // Generator not called again

	// Different key - should call generator
	result3 := cache.MemoizeInferSchema("key2", generator)
	assert.Equal(t, expectedResult, result3)
	assert.Equal(t, 2, callCount)
}

func TestSchemaCache_ClearMemoCache(t *testing.T) {
	cache := NewSchemaCache()

	// Add some entries to memo cache
	cache.MemoizeInferSchema("key1", func() map[string]any {
		return map[string]any{"test": "value1"}
	})
	cache.MemoizeInferSchema("key2", func() map[string]any {
		return map[string]any{"test": "value2"}
	})

	assert.Len(t, cache.memoCache, 2)

	// Clear cache
	cache.ClearMemoCache()

	assert.Empty(t, cache.memoCache)
}

func TestSchemaCache_SaveToFile_LoadFromFile(t *testing.T) {
	cache := NewSchemaCache()
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, "cache.json")

	// Add some entries
	entry1 := &CacheEntry{
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"test": map[string]any{"type": "string"},
			},
		},
		ModTime:  time.Now(),
		Checksum: "abc123",
	}

	entry2 := &CacheEntry{
		Schema: map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "number"},
		},
		ModTime:  time.Now().Add(-1 * time.Hour),
		Checksum: "def456",
	}

	cache.mu.Lock()
	cache.entries["file1.yaml"] = entry1
	cache.entries["file2.yaml"] = entry2
	cache.mu.Unlock()

	// Save to file
	err := cache.SaveToFile(cacheFile)
	assert.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(cacheFile)
	assert.NoError(t, err)

	// Create new cache and load from file
	newCache := NewSchemaCache()
	err = newCache.LoadFromFile(cacheFile)
	assert.NoError(t, err)

	// Verify entries were loaded
	assert.Len(t, newCache.entries, 2)

	// Check entry1
	loadedEntry1, ok := newCache.entries["file1.yaml"]
	assert.True(t, ok)
	assert.Equal(t, entry1.Schema, loadedEntry1.Schema)
	assert.Equal(t, entry1.Checksum, loadedEntry1.Checksum)
	assert.WithinDuration(t, entry1.ModTime, loadedEntry1.ModTime, time.Second)

	// Check entry2
	loadedEntry2, ok := newCache.entries["file2.yaml"]
	assert.True(t, ok)
	assert.Equal(t, entry2.Schema, loadedEntry2.Schema)
	assert.Equal(t, entry2.Checksum, loadedEntry2.Checksum)
	assert.WithinDuration(t, entry2.ModTime, loadedEntry2.ModTime, time.Second)
}

func TestSchemaCache_LoadFromFile_NonExistent(t *testing.T) {
	cache := NewSchemaCache()

	// Loading from non-existent file should not error
	err := cache.LoadFromFile("/tmp/non-existent-cache.json")
	assert.NoError(t, err)
	assert.Empty(t, cache.entries)
}

func TestSchemaCache_LoadFromFile_InvalidJSON(t *testing.T) {
	cache := NewSchemaCache()
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, "invalid.json")

	// Write invalid JSON
	err := os.WriteFile(cacheFile, []byte("invalid json content"), 0644)
	require.NoError(t, err)

	// Loading should error
	err = cache.LoadFromFile(cacheFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal cache")
}

func TestSchemaCache_SaveToFile_CreateDirectory(t *testing.T) {
	cache := NewSchemaCache()
	tmpDir := t.TempDir()
	nestedCacheFile := filepath.Join(tmpDir, "nested", "dir", "cache.json")

	// Add an entry
	cache.mu.Lock()
	cache.entries["test.yaml"] = &CacheEntry{
		Schema:   map[string]any{"type": "string"},
		ModTime:  time.Now(),
		Checksum: "test123",
	}
	cache.mu.Unlock()

	// Save should create directories
	err := cache.SaveToFile(nestedCacheFile)
	assert.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(nestedCacheFile)
	assert.NoError(t, err)
}

func TestGenerateMemoKey(t *testing.T) {
	tests := []struct {
		name     string
		val      any
		defVal   any
		sameKey  bool
		otherVal any
		otherDef any
	}{
		{
			name:     "same inputs produce same key",
			val:      map[string]any{"test": "value"},
			defVal:   map[string]any{"test": "default"},
			sameKey:  true,
			otherVal: map[string]any{"test": "value"},
			otherDef: map[string]any{"test": "default"},
		},
		{
			name:     "different values produce different keys",
			val:      map[string]any{"test": "value1"},
			defVal:   map[string]any{"test": "default"},
			sameKey:  false,
			otherVal: map[string]any{"test": "value2"},
			otherDef: map[string]any{"test": "default"},
		},
		{
			name:     "different defaults produce different keys",
			val:      map[string]any{"test": "value"},
			defVal:   map[string]any{"test": "default1"},
			sameKey:  false,
			otherVal: map[string]any{"test": "value"},
			otherDef: map[string]any{"test": "default2"},
		},
		{
			name:     "different types produce different keys",
			val:      "string value",
			defVal:   "string default",
			sameKey:  false,
			otherVal: 123,
			otherDef: 456,
		},
		{
			name:     "nil values",
			val:      nil,
			defVal:   nil,
			sameKey:  true,
			otherVal: nil,
			otherDef: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key1 := GenerateMemoKey(tt.val, tt.defVal)
			key2 := GenerateMemoKey(tt.otherVal, tt.otherDef)

			// Keys should be hex strings
			assert.Regexp(t, "^[a-f0-9]{64}$", key1)
			assert.Regexp(t, "^[a-f0-9]{64}$", key2)

			if tt.sameKey {
				assert.Equal(t, key1, key2)
			} else {
				assert.NotEqual(t, key1, key2)
			}
		})
	}
}

func TestCalculateFileChecksum(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with a file
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("test content")
	err := os.WriteFile(testFile, content, 0644)
	require.NoError(t, err)

	checksum, err := calculateFileChecksum(testFile)
	assert.NoError(t, err)
	assert.NotEmpty(t, checksum)
	assert.Regexp(t, "^[a-f0-9]{64}$", checksum)

	// Test with non-existent file
	_, err = calculateFileChecksum("/tmp/non-existent-file")
	assert.Error(t, err)

	// Test that same content produces same checksum
	checksum2, err := calculateFileChecksum(testFile)
	assert.NoError(t, err)
	assert.Equal(t, checksum, checksum2)

	// Test that different content produces different checksum
	err = os.WriteFile(testFile, []byte("different content"), 0644)
	require.NoError(t, err)

	checksum3, err := calculateFileChecksum(testFile)
	assert.NoError(t, err)
	assert.NotEqual(t, checksum, checksum3)
}

func TestSchemaCache_ConcurrentAccess(t *testing.T) {
	cache := NewSchemaCache()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.yaml")

	// Create test file
	err := os.WriteFile(testFile, []byte("test: value"), 0644)
	require.NoError(t, err)

	// Test concurrent access to GetOrGenerate
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := cache.GetOrGenerate(testFile, func() (map[string]any, error) {
				return map[string]any{"test": "value"}, nil
			})
			assert.NoError(t, err)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent access to MemoizeInferSchema
	for i := 0; i < 10; i++ {
		go func(n int) {
			key := fmt.Sprintf("key%d", n)
			cache.MemoizeInferSchema(key, func() map[string]any {
				return map[string]any{"test": n}
			})
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
