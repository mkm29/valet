package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeFile is a helper function for tests
func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

func TestInferSchema(t *testing.T) {
	tests := []struct {
		name        string
		value       any
		defaultVal  any
		expected    map[string]any
		description string
	}{
		{
			name:  "nil value",
			value: nil,
			expected: map[string]any{
				"type":    []string{"string", "null"},
				"default": nil,
			},
			description: "nil values should return string|null type",
		},
		{
			name:  "boolean value",
			value: true,
			expected: map[string]any{
				"type":    "boolean",
				"default": true,
			},
		},
		{
			name:  "integer value",
			value: 42,
			expected: map[string]any{
				"type":    "integer",
				"default": 42,
			},
		},
		{
			name:  "float as integer",
			value: 42.0,
			expected: map[string]any{
				"type":    "integer",
				"default": int64(42),
			},
			description: "floats that are whole numbers should be integers",
		},
		{
			name:  "float value",
			value: 42.5,
			expected: map[string]any{
				"type":    "number",
				"default": 42.5,
			},
		},
		{
			name:  "empty string",
			value: "",
			expected: map[string]any{
				"type":    []string{"string", "null"},
				"default": nil,
			},
		},
		{
			name:  "null string",
			value: "null",
			expected: map[string]any{
				"type":    []string{"string", "null"},
				"default": nil,
			},
		},
		{
			name:  "nil string",
			value: "<nil>",
			expected: map[string]any{
				"type":    []string{"string", "null"},
				"default": nil,
			},
		},
		{
			name:  "regular string",
			value: "hello world",
			expected: map[string]any{
				"type":    "string",
				"default": "hello world",
			},
		},
		{
			name: "simple map",
			value: map[string]any{
				"key1": "value1",
				"key2": 123,
			},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key1": map[string]any{
						"type":    "string",
						"default": "value1",
					},
					"key2": map[string]any{
						"type":    "integer",
						"default": 123,
					},
				},
				"default": map[string]any{
					"key1": "value1",
					"key2": 123,
				},
			},
		},
		{
			name: "nested map",
			value: map[string]any{
				"outer": map[string]any{
					"inner": "value",
				},
			},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"outer": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"inner": map[string]any{
								"type":    "string",
								"default": "value",
							},
						},
						"default": map[string]any{
							"inner": "value",
						},
					},
				},
				"default": map[string]any{
					"outer": map[string]any{
						"inner": "value",
					},
				},
			},
		},
		{
			name:  "empty array",
			value: []any{},
			expected: map[string]any{
				"type":    "array",
				"items":   map[string]any{}, // Empty schema for empty arrays
				"default": []any{},
			},
		},
		{
			name:  "string array",
			value: []any{"a", "b", "c"}, // inferSchema expects []any
			expected: map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":    "string",
					"default": "a", // First item becomes default in items schema
				},
				"default": []any{"a", "b", "c"},
			},
		},
		{
			name:  "mixed type array",
			value: []any{"string", 123, true},
			expected: map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":    "string",
					"default": "string", // Only first item is considered
				},
				"default": []any{"string", 123, true},
			},
		},
		{
			name: "map with nil value",
			value: map[string]any{
				"key1": "value1",
				"key2": nil,
			},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key1": map[string]any{
						"type":    "string",
						"default": "value1",
					},
				},
				"default": map[string]any{
					"key1": "value1",
				},
			},
			description: "nil values in maps should be filtered out",
		},
		{
			name:       "with default value override",
			value:      "current",
			defaultVal: "default",
			expected: map[string]any{
				"type":    "string",
				"default": "default",
			},
			description: "default value parameter should override inferred default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferSchema(tt.value, tt.defaultVal)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestIsEmptyValue(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		{"nil", nil, true},
		{"empty string", "", true},
		{"empty map", map[string]any{}, true},
		{"empty slice", []any{}, true},
		{"empty array", [0]int{}, true},
		{"zero int", 0, true},
		{"zero float", 0.0, true},
		{"false bool", false, true},
		{"non-empty string", "hello", false},
		{"non-zero int", 42, false},
		{"non-zero float", 3.14, false},
		{"true bool", true, false},
		{"non-empty map", map[string]any{"key": "value"}, false},
		{"non-empty slice", []int{1, 2, 3}, false},
		{"nil pointer", (*string)(nil), true},
		{"empty struct", struct{}{}, true},
		{"struct with fields", struct{ Name string }{Name: "test"}, false},
		{"empty interface with nil", interface{}(nil), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEmptyValue(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProcessProperties(t *testing.T) {
	tests := []struct {
		name           string
		schema         map[string]any
		defaults       map[string]any
		expectedSchema map[string]any
		description    string
	}{
		{
			name: "removes empty defaults from required",
			schema: map[string]any{
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
					"age":  map[string]any{"type": "integer"},
				},
				"required": []string{"name", "age"},
			},
			defaults: map[string]any{
				"name": "",    // empty string
				"age":  25,    // has value
			},
			expectedSchema: map[string]any{
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
					"age":  map[string]any{"type": "integer"},
				},
				"required": []string{"age"}, // name removed because empty default
			},
		},
		{
			name: "removes disabled components from required",
			schema: map[string]any{
				"properties": map[string]any{
					"feature1": map[string]any{"type": "object"},
					"feature2": map[string]any{"type": "object"},
				},
				"required": []string{"feature1", "feature2"},
			},
			defaults: map[string]any{
				"feature1": map[string]any{"enabled": false}, // disabled
				"feature2": map[string]any{"enabled": true},  // enabled
			},
			expectedSchema: map[string]any{
				"properties": map[string]any{
					"feature1": map[string]any{"type": "object"},
					"feature2": map[string]any{"type": "object"},
				},
				"required": []string{"feature2"}, // feature1 removed
			},
		},
		{
			name: "removes required when object has enabled=false",
			schema: map[string]any{
				"properties": map[string]any{
					"field1": map[string]any{"type": "string"},
				},
				"required": []string{"field1"},
			},
			defaults: map[string]any{
				"enabled": false, // whole object disabled
			},
			expectedSchema: map[string]any{
				"properties": map[string]any{
					"field1": map[string]any{"type": "string"},
				},
				// required removed entirely
			},
		},
		{
			name: "no required field to process",
			schema: map[string]any{
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
				},
			},
			defaults: map[string]any{},
			expectedSchema: map[string]any{
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
				},
			},
		},
		{
			name: "processes nested objects recursively",
			schema: map[string]any{
				"properties": map[string]any{
					"config": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"setting": map[string]any{"type": "string"},
						},
						"required": []string{"setting"},
					},
				},
			},
			defaults: map[string]any{
				"config": map[string]any{
					"setting": "", // empty
				},
			},
			expectedSchema: map[string]any{
				"properties": map[string]any{
					"config": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"setting": map[string]any{"type": "string"},
						},
						// required removed because setting has empty default
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a deep copy to avoid modifying test data
			schemaCopy := make(map[string]any)
			for k, v := range tt.schema {
				schemaCopy[k] = v
			}
			
			// processProperties modifies the schema in place
			processProperties(schemaCopy, tt.defaults)
			
			// Check the modified schema
			assert.Equal(t, tt.expectedSchema, schemaCopy, tt.description)
		})
	}
}

func TestGenerateCommand_ErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func() (string, func())
		args          []string
		expectedError string
	}{
		{
			name: "missing values file",
			setupFunc: func() (string, func()) {
				tmpDir := t.TempDir()
				return tmpDir, func() {}
			},
			args:          []string{"non-existent.yaml"},
			expectedError: "no such file or directory",
		},
		{
			name: "invalid YAML",
			setupFunc: func() (string, func()) {
				tmpDir := t.TempDir()
				valuesFile := tmpDir + "/invalid.yaml"
				require.NoError(t, writeFile(valuesFile, []byte("invalid: yaml: content\n  bad indentation")))
				return tmpDir, func() {}
			},
			args:          []string{"invalid.yaml"},
			expectedError: "yaml:",
		},
		{
			name: "invalid JSON",
			setupFunc: func() (string, func()) {
				tmpDir := t.TempDir()
				valuesFile := tmpDir + "/invalid.json"
				require.NoError(t, writeFile(valuesFile, []byte(`{"invalid": json content}`)))
				return tmpDir, func() {}
			},
			args:          []string{"invalid.json"},
			expectedError: "invalid character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, cleanup := tt.setupFunc()
			defer cleanup()

			// Change to temp directory
			oldDir, err := os.Getwd()
			require.NoError(t, err)
			require.NoError(t, os.Chdir(tmpDir))
			defer os.Chdir(oldDir)

			cmd := NewGenerateCmd()
			cmd.SetArgs(tt.args)
			err = cmd.Execute()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}