package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// Benchmarks for inferSchema function
func BenchmarkInferSchema_SimpleTypes(b *testing.B) {
	values := []any{
		"string value",
		123,
		3.14,
		true,
		nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, v := range values {
			_ = inferSchema(v, nil)
		}
	}
}

func BenchmarkInferSchema_Map(b *testing.B) {
	value := map[string]any{
		"key1": "value1",
		"key2": 123,
		"key3": true,
		"nested": map[string]any{
			"inner1": "value",
			"inner2": 456,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = inferSchema(value, nil)
	}
}

func BenchmarkInferSchema_Array(b *testing.B) {
	value := []any{
		"string",
		123,
		true,
		map[string]any{"key": "value"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = inferSchema(value, nil)
	}
}

func BenchmarkInferSchema_DeepNesting(b *testing.B) {
	// Create a deeply nested structure
	value := make(map[string]any)
	current := value
	for i := 0; i < 10; i++ {
		next := make(map[string]any)
		current["level"] = next
		current = next
	}
	current["value"] = "deep"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = inferSchema(value, nil)
	}
}

// Benchmarks for processProperties function
func BenchmarkProcessProperties(b *testing.B) {
	props := map[string]any{
		"prop1": map[string]any{"type": "string"},
		"prop2": map[string]any{"type": "integer"},
		"prop3": map[string]any{"type": "boolean"},
		"nested": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"inner1": map[string]any{"type": "string"},
				"inner2": map[string]any{"type": "number"},
			},
		},
	}

	defaults := map[string]any{
		"prop1": "default",
		"nested": map[string]any{
			"inner1": "default",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Make a copy to avoid mutation affecting benchmark
		propsCopy := make(map[string]any)
		for k, v := range props {
			propsCopy[k] = v
		}
		processProperties(propsCopy, defaults)
	}
}

// Benchmark the entire generate command flow
func BenchmarkGenerateCommand_SmallFile(b *testing.B) {
	tmpDir := b.TempDir()
	
	values := map[string]any{
		"name":      "test-app",
		"namespace": "default",
		"replicas":  3,
		"image":     "nginx:latest",
	}

	valuesFile := filepath.Join(tmpDir, "values.yaml")
	data, _ := yaml.Marshal(values)
	os.WriteFile(valuesFile, data, 0644)

	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := NewGenerateCmd()
		cmd.SetArgs([]string{valuesFile})
		cmd.Execute()
	}
}

func BenchmarkGenerateCommand_LargeFile(b *testing.B) {
	tmpDir := b.TempDir()
	
	// Create a large values structure
	values := make(map[string]any)
	for i := 0; i < 100; i++ {
		values[fmt.Sprintf("service%d", i)] = map[string]any{
			"enabled":   i%2 == 0,
			"replicas":  i % 5,
			"image":     fmt.Sprintf("app%d:v1.0.0", i),
			"port":      8000 + i,
			"resources": map[string]any{
				"limits": map[string]any{
					"cpu":    fmt.Sprintf("%dm", 100+i*10),
					"memory": fmt.Sprintf("%dMi", 128+i*16),
				},
				"requests": map[string]any{
					"cpu":    fmt.Sprintf("%dm", 50+i*5),
					"memory": fmt.Sprintf("%dMi", 64+i*8),
				},
			},
			"env": []map[string]any{
				{"name": "ENV1", "value": fmt.Sprintf("value%d", i)},
				{"name": "ENV2", "value": fmt.Sprintf("value%d", i*2)},
			},
		}
	}

	valuesFile := filepath.Join(tmpDir, "values.yaml")
	data, _ := yaml.Marshal(values)
	os.WriteFile(valuesFile, data, 0644)

	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := NewGenerateCmd()
		cmd.SetArgs([]string{valuesFile})
		cmd.Execute()
	}
}

// Benchmark JSON vs YAML parsing
func BenchmarkParsing_YAML(b *testing.B) {
	data := []byte(`
name: test-app
replicas: 3
resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 50m
    memory: 64Mi
`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var values map[string]any
		yaml.Unmarshal(data, &values)
	}
}

func BenchmarkParsing_JSON(b *testing.B) {
	data := []byte(`{
  "name": "test-app",
  "replicas": 3,
  "resources": {
    "limits": {
      "cpu": "100m",
      "memory": "128Mi"
    },
    "requests": {
      "cpu": "50m",
      "memory": "64Mi"
    }
  }
}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var values map[string]any
		json.Unmarshal(data, &values)
	}
}