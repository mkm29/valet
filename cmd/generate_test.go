package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeepMerge(t *testing.T) {
	tests := []struct {
		name     string
		a        map[string]any
		b        map[string]any
		expected map[string]any
	}{
		{
			name:     "empty maps",
			a:        map[string]any{},
			b:        map[string]any{},
			expected: map[string]any{},
		},
		{
			name: "merge simple values",
			a: map[string]any{
				"key1": "value1",
				"key2": 42,
			},
			b: map[string]any{
				"key2": 100,
				"key3": true,
			},
			expected: map[string]any{
				"key1": "value1",
				"key2": 100,
				"key3": true,
			},
		},
		{
			name: "merge nested maps",
			a: map[string]any{
				"config": map[string]any{
					"host": "localhost",
					"port": 8080,
				},
			},
			b: map[string]any{
				"config": map[string]any{
					"port":     9090,
					"protocol": "https",
				},
			},
			expected: map[string]any{
				"config": map[string]any{
					"host":     "localhost",
					"port":     9090,
					"protocol": "https",
				},
			},
		},
		{
			name: "merge with non-map values",
			a: map[string]any{
				"value": "string",
			},
			b: map[string]any{
				"value": map[string]any{"nested": "value"},
			},
			expected: map[string]any{
				"value": map[string]any{"nested": "value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deepMerge(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInferSchema(t *testing.T) {
	tests := []struct {
		name       string
		val        any
		defaultVal any
		expected   map[string]any
	}{
		{
			name:       "string value",
			val:        "test",
			defaultVal: "test",
			expected: map[string]any{
				"type":    "string",
				"default": "test",
			},
		},
		{
			name:       "integer value",
			val:        42,
			defaultVal: 42,
			expected: map[string]any{
				"type":    "integer",
				"default": 42,
			},
		},
		{
			name:       "boolean value",
			val:        true,
			defaultVal: true,
			expected: map[string]any{
				"type":    "boolean",
				"default": true,
			},
		},
		{
			name:       "float as integer",
			val:        42.0,
			defaultVal: 42.0,
			expected: map[string]any{
				"type":    "integer",
				"default": int64(42),
			},
		},
		{
			name:       "float value",
			val:        42.5,
			defaultVal: 42.5,
			expected: map[string]any{
				"type":    "number",
				"default": 42.5,
			},
		},
		{
			name:       "array value",
			val:        []any{"a", "b", "c"},
			defaultVal: []any{"a", "b", "c"},
			expected: map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":    "string",
					"default": "a",
				},
				"default": []any{"a", "b", "c"},
			},
		},
		{
			name:       "empty array",
			val:        []any{},
			defaultVal: []any{},
			expected: map[string]any{
				"type":    "array",
				"items":   map[string]any{},
				"default": []any{},
			},
		},
		{
			name: "object with required fields",
			val: map[string]any{
				"field1": "value1",
				"field2": 42,
			},
			defaultVal: map[string]any{
				"field1": "value1",
				"field2": 42,
			},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"field1": map[string]any{
						"type":    "string",
						"default": "value1",
					},
					"field2": map[string]any{
						"type":    "integer",
						"default": 42,
					},
				},
				"default": map[string]any{
					"field1": "value1",
					"field2": 42,
				},
				"required": []string{"field1", "field2"},
			},
		},
		{
			name: "object with empty string field",
			val: map[string]any{
				"field1": "",
			},
			defaultVal: map[string]any{
				"field1": "",
			},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"field1": map[string]any{
						"type":    []string{"string", "null"},
						"default": nil,
					},
				},
				"default": map[string]any{
					"field1": "",
				},
			},
		},
		{
			name: "object with disabled component",
			val: map[string]any{
				"component": map[string]any{
					"enabled": false,
					"field":   "value",
				},
			},
			defaultVal: map[string]any{
				"component": map[string]any{
					"enabled": false,
					"field":   "value",
				},
			},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"component": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"enabled": map[string]any{
								"type":    "boolean",
								"default": false,
							},
							"field": map[string]any{
								"type":    "string",
								"default": "value",
							},
						},
						"default": map[string]any{
							"enabled": false,
							"field":   "value",
						},
						"required": []string{"enabled", "field"},
					},
				},
				"default": map[string]any{
					"component": map[string]any{
						"enabled": false,
						"field":   "value",
					},
				},
			},
		},
		{
			name:       "null string",
			val:        "null",
			defaultVal: "null",
			expected: map[string]any{
				"type":    []string{"string", "null"},
				"default": nil,
			},
		},
		{
			name:       "nil value",
			val:        nil,
			defaultVal: nil,
			expected: map[string]any{
				"type":    []string{"string", "null"},
				"default": nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferSchema(tt.val, tt.defaultVal)

			// For the "object with disabled component" test, we need to handle the required array separately
			// because map iteration order is not guaranteed
			if tt.name == "object with disabled component" {
				// Check everything except the required array
				resultComp := result["properties"].(map[string]any)["component"].(map[string]any)
				expectedComp := tt.expected["properties"].(map[string]any)["component"].(map[string]any)

				// Extract and check required arrays separately
				resultRequired := resultComp["required"].([]string)
				expectedRequired := expectedComp["required"].([]string)
				assert.ElementsMatch(t, expectedRequired, resultRequired)

				// Remove required from both before comparing the rest
				delete(resultComp, "required")
				delete(expectedComp, "required")

				// Now compare the rest
				assert.Equal(t, tt.expected, result)

				// Put required back for completeness
				resultComp["required"] = resultRequired
				expectedComp["required"] = expectedRequired
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestConvertToStringKeyMap(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "simple string",
			input:    "test",
			expected: "test",
		},
		{
			name:     "simple number",
			input:    42,
			expected: 42,
		},
		{
			name: "map with interface keys",
			input: map[interface{}]interface{}{
				"key1": "value1",
				"key2": 42,
			},
			expected: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
			},
		},
		{
			name: "nested maps",
			input: map[interface{}]interface{}{
				"outer": map[interface{}]interface{}{
					"inner": "value",
				},
			},
			expected: map[string]interface{}{
				"outer": map[string]interface{}{
					"inner": "value",
				},
			},
		},
		{
			name: "array with maps",
			input: []interface{}{
				map[interface{}]interface{}{
					"key": "value",
				},
			},
			expected: []interface{}{
				map[string]interface{}{
					"key": "value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToStringKeyMap(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadYAML(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		filename    string
		content     string
		expected    map[string]any
		expectError bool
	}{
		{
			name:     "valid yaml",
			filename: "valid.yaml",
			content: `key1: value1
key2: 42
key3:
  nested: true`,
			expected: map[string]any{
				"key1": "value1",
				"key2": 42,
				"key3": map[string]any{
					"nested": true,
				},
			},
			expectError: false,
		},
		{
			name:        "empty file",
			filename:    "empty.yaml",
			content:     "",
			expected:    map[string]any{},
			expectError: false,
		},
		{
			name:        "invalid yaml",
			filename:    "invalid.yaml",
			content:     "key1: value1\n- invalid indentation\n    - bad yaml",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "non-existent file",
			filename:    "nonexistent.yaml",
			content:     "",
			expected:    map[string]any{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filepath := filepath.Join(tmpDir, tt.filename)

			// Create file if not testing non-existent case
			if tt.name != "non-existent file" {
				err := os.WriteFile(filepath, []byte(tt.content), 0644)
				require.NoError(t, err)
			}

			result, err := loadYAML(filepath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGenerate(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test values.yaml
	valuesContent := `replicaCount: 3
image:
  repository: nginx
  tag: latest
service:
  type: ClusterIP
  port: 80
ingress:
  enabled: false
  hosts: []`

	err := os.WriteFile(filepath.Join(tmpDir, "values.yaml"), []byte(valuesContent), 0644)
	require.NoError(t, err)

	// Test without overrides
	msg, err := Generate(tmpDir, "")
	assert.NoError(t, err)
	assert.Contains(t, msg, "Generated")
	assert.Contains(t, msg, "values.schema.json")

	// Verify schema file was created
	schemaPath := filepath.Join(tmpDir, "values.schema.json")
	assert.FileExists(t, schemaPath)

	// Verify schema content
	schemaData, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	var schema map[string]any
	err = json.Unmarshal(schemaData, &schema)
	require.NoError(t, err)

	assert.Equal(t, "http://json-schema.org/schema#", schema["$schema"])
	assert.Equal(t, "object", schema["type"])
	assert.NotNil(t, schema["properties"])

	// Test with overrides
	overridesContent := `replicaCount: 5
service:
  port: 8080
newField: value`

	err = os.WriteFile(filepath.Join(tmpDir, "overrides.yaml"), []byte(overridesContent), 0644)
	require.NoError(t, err)

	msg, err = Generate(tmpDir, "overrides.yaml")
	assert.NoError(t, err)
	assert.Contains(t, msg, "merging")
	assert.Contains(t, msg, "overrides.yaml")
}

func TestGenerateErrors(t *testing.T) {
	tmpDir := t.TempDir()

	// Test missing values.yaml
	_, err := Generate(tmpDir, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no values.yaml or values.yml found")

	// Test with values.yml instead
	err = os.WriteFile(filepath.Join(tmpDir, "values.yml"), []byte("key: value"), 0644)
	require.NoError(t, err)

	msg, err := Generate(tmpDir, "")
	assert.NoError(t, err)
	assert.Contains(t, msg, "Generated")

	// Test with non-existent overrides file - should not error since loadYAML returns empty map
	tmpDir2 := t.TempDir()
	err = os.WriteFile(filepath.Join(tmpDir2, "values.yaml"), []byte("key: value"), 0644)
	require.NoError(t, err)

	// Non-existent overrides file should not cause error
	msg, err = Generate(tmpDir2, "nonexistent.yaml")
	assert.NoError(t, err)
	assert.Contains(t, msg, "Generated")
	assert.Contains(t, msg, "merging")
}

func TestIsEmptyValue(t *testing.T) {
	tests := []struct {
		name     string
		val      any
		expected bool
	}{
		{"nil", nil, true},
		{"empty string", "", true},
		{"non-empty string", "test", false},
		{"empty array", []any{}, true},
		{"non-empty array", []any{"a"}, false},
		{"empty map", map[string]any{}, true},
		{"non-empty map", map[string]any{"key": "value"}, false},
		{"empty interface map", map[interface{}]interface{}{}, true},
		{"non-empty interface map", map[interface{}]interface{}{"key": "value"}, false},
		{"number zero", 0, false},
		{"number non-zero", 42, false},
		{"boolean false", false, false},
		{"boolean true", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEmptyValue(tt.val)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCountSchemaFields(t *testing.T) {
	tests := []struct {
		name     string
		schema   map[string]any
		expected int
	}{
		{
			name:     "empty schema",
			schema:   map[string]any{},
			expected: 0,
		},
		{
			name: "simple properties",
			schema: map[string]any{
				"properties": map[string]any{
					"field1": map[string]any{"type": "string"},
					"field2": map[string]any{"type": "integer"},
				},
			},
			expected: 2,
		},
		{
			name: "nested properties",
			schema: map[string]any{
				"properties": map[string]any{
					"field1": map[string]any{"type": "string"},
					"nested": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"subfield1": map[string]any{"type": "string"},
							"subfield2": map[string]any{"type": "integer"},
						},
					},
				},
			},
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countSchemaFields(tt.schema)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCleanupRequiredFields(t *testing.T) {
	tests := []struct {
		name     string
		schema   map[string]any
		defaults map[string]any
		expected map[string]any
	}{
		{
			name: "remove empty string from required",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"field1": map[string]any{"type": "string"},
					"field2": map[string]any{"type": "string"},
				},
				"required": []string{"field1", "field2"},
			},
			defaults: map[string]any{
				"field1": "value",
				"field2": "",
			},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"field1": map[string]any{"type": "string"},
					"field2": map[string]any{"type": "string"},
				},
				"required": []string{"field1"},
			},
		},
		{
			name: "remove disabled component from required",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"component": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"enabled": map[string]any{"type": "boolean"},
						},
					},
				},
				"required": []string{"component"},
			},
			defaults: map[string]any{
				"component": map[string]any{
					"enabled": false,
				},
			},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"component": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"enabled": map[string]any{"type": "boolean"},
						},
					},
				},
			},
		},
		{
			name: "remove all required if component disabled",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"enabled": map[string]any{"type": "boolean"},
					"field1":  map[string]any{"type": "string"},
				},
				"required": []string{"enabled", "field1"},
			},
			defaults: map[string]any{
				"enabled": false,
				"field1":  "value",
			},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"enabled": map[string]any{"type": "boolean"},
					"field1":  map[string]any{"type": "string"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupRequiredFields(tt.schema, tt.defaults)
			assert.Equal(t, tt.expected, tt.schema)
		})
	}
}

func TestNewGenerateCmd(t *testing.T) {
	cmd := NewGenerateCmd()

	assert.NotNil(t, cmd)
	assert.Equal(t, "generate <context-dir>", cmd.Use)
	assert.Equal(t, "Generate JSON Schema from values.yaml", cmd.Short)
	assert.True(t, cmd.SilenceUsage)
	assert.NotNil(t, cmd.RunE)

	// Check flags
	overridesFlag := cmd.Flags().Lookup("overrides")
	assert.NotNil(t, overridesFlag)
	assert.Equal(t, "f", overridesFlag.Shorthand)
	assert.Equal(t, "", overridesFlag.DefValue)
}

func TestGenerateCmdExecution(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test values.yaml
	valuesContent := `app:
  name: test-app
  version: 1.0.0`

	err := os.WriteFile(filepath.Join(tmpDir, "values.yaml"), []byte(valuesContent), 0644)
	require.NoError(t, err)

	// Test command execution
	cmd := NewGenerateCmd()
	cmd.SetArgs([]string{tmpDir})

	err = cmd.Execute()
	assert.NoError(t, err)

	// Verify schema was created
	schemaPath := filepath.Join(tmpDir, "values.schema.json")
	assert.FileExists(t, schemaPath)

	// Test with missing overrides file
	cmd = NewGenerateCmd()
	cmd.SetArgs([]string{tmpDir, "--overrides", "missing.yaml"})

	err = cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "overrides file missing.yaml not found")
}
