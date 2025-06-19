package tests

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/mkm29/valet/cmd"
	"gopkg.in/yaml.v3"
)

func (ts *ValetTestSuite) TestEndToEnd_YAMLToJSONSchema() {
	// Test complete workflow: YAML values -> JSON Schema generation
	tmpDir := ts.T().TempDir()
	
	// Create a comprehensive values.yaml file
	valuesContent := `
# Global settings
global:
  image:
    repository: myapp
    tag: v1.0.0
    pullPolicy: IfNotPresent
  replicas: 3
  enabled: true

# Service configuration
service:
  type: ClusterIP
  port: 8080
  targetPort: 8080
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: "nlb"

# Ingress settings
ingress:
  enabled: false
  className: nginx
  hosts:
    - host: example.com
      paths:
        - path: /
          pathType: Prefix
  tls: []

# Resources
resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 50m
    memory: 64Mi

# Environment variables
env:
  - name: LOG_LEVEL
    value: info
  - name: DB_HOST
    value: postgres.default.svc.cluster.local

# Empty and null values
emptyString: ""
nullValue: null
emptyObject: {}
emptyArray: []
`

	valuesFile := filepath.Join(tmpDir, "values.yaml")
	ts.Require().NoError(os.WriteFile(valuesFile, []byte(valuesContent), 0644))

	// Change to temp directory
	oldDir, err := os.Getwd()
	ts.Require().NoError(err)
	ts.Require().NoError(os.Chdir(tmpDir))
	defer os.Chdir(oldDir)

	// Run generate command - it expects a directory, not a file
	rootCmd := cmd.NewRootCmd()
	rootCmd.SetArgs([]string{"generate", tmpDir})
	ts.Require().NoError(rootCmd.Execute())

	// Verify schema file was created
	schemaFile := filepath.Join(tmpDir, "values.schema.json")
	ts.Require().FileExists(schemaFile)

	// Read and validate the schema
	schemaData, err := os.ReadFile(schemaFile)
	ts.Require().NoError(err)

	var schema map[string]any
	ts.Require().NoError(json.Unmarshal(schemaData, &schema))

	// Validate schema structure
	ts.Equal("http://json-schema.org/schema#", schema["$schema"])
	ts.Equal("object", schema["type"])
	ts.NotNil(schema["properties"])

	props := schema["properties"].(map[string]any)
	
	// Check global properties
	global := props["global"].(map[string]any)
	ts.Equal("object", global["type"])
	globalProps := global["properties"].(map[string]any)
	
	// Check nested image properties
	image := globalProps["image"].(map[string]any)
	ts.Equal("object", image["type"])
	imageProps := image["properties"].(map[string]any)
	
	repository := imageProps["repository"].(map[string]any)
	ts.Equal("string", repository["type"])
	ts.Equal("myapp", repository["default"])
	
	// Check array handling
	env := props["env"].(map[string]any)
	ts.Equal("array", env["type"])
	ts.NotNil(env["items"])
	
	// Check empty value handling
	emptyStr := props["emptyString"].(map[string]any)
	ts.Contains(emptyStr["type"], "string")
	
	nullVal := props["nullValue"].(map[string]any)
	ts.Contains(nullVal["type"], "null")
}

func (ts *ValetTestSuite) TestEndToEnd_JSONToJSONSchema() {
	// Test JSON values file to schema generation
	tmpDir := ts.T().TempDir()
	
	valuesContent := `{
  "app": {
    "name": "test-app",
    "version": "1.0.0",
    "ports": [8080, 8443],
    "features": {
      "logging": true,
      "metrics": false
    }
  },
  "database": {
    "host": "localhost",
    "port": 5432,
    "ssl": null
  }
}`

	valuesFile := filepath.Join(tmpDir, "values.json")
	ts.Require().NoError(os.WriteFile(valuesFile, []byte(valuesContent), 0644))

	oldDir, err := os.Getwd()
	ts.Require().NoError(err)
	ts.Require().NoError(os.Chdir(tmpDir))
	defer os.Chdir(oldDir)

	rootCmd := cmd.NewRootCmd()
	rootCmd.SetArgs([]string{"generate", tmpDir})
	ts.Require().NoError(rootCmd.Execute())

	schemaFile := filepath.Join(tmpDir, "values.schema.json")
	ts.Require().FileExists(schemaFile)

	schemaData, err := os.ReadFile(schemaFile)
	ts.Require().NoError(err)

	var schema map[string]any
	ts.Require().NoError(json.Unmarshal(schemaData, &schema))

	// Validate specific JSON handling
	props := schema["properties"].(map[string]any)
	app := props["app"].(map[string]any)
	appProps := app["properties"].(map[string]any)
	
	ports := appProps["ports"].(map[string]any)
	ts.Equal("array", ports["type"])
	items := ports["items"].(map[string]any)
	ts.Equal("integer", items["type"])
}

func (ts *ValetTestSuite) TestEndToEnd_WithDefaults() {
	// Test schema generation with default values
	tmpDir := ts.T().TempDir()
	
	valuesContent := `
name: my-app
namespace: default
`
	defaultsContent := `
name: default-app
namespace: default
replicas: 1
resources:
  limits:
    cpu: "1"
    memory: "1Gi"
`

	valuesFile := filepath.Join(tmpDir, "values.yaml")
	defaultsFile := filepath.Join(tmpDir, "defaults.yaml")
	
	ts.Require().NoError(os.WriteFile(valuesFile, []byte(valuesContent), 0644))
	ts.Require().NoError(os.WriteFile(defaultsFile, []byte(defaultsContent), 0644))

	oldDir, err := os.Getwd()
	ts.Require().NoError(err)
	ts.Require().NoError(os.Chdir(tmpDir))
	defer os.Chdir(oldDir)

	rootCmd := cmd.NewRootCmd()
	rootCmd.SetArgs([]string{"generate", tmpDir, "--overrides", "defaults.yaml"})
	ts.Require().NoError(rootCmd.Execute())

	schemaFile := filepath.Join(tmpDir, "values.schema.json")
	schemaData, err := os.ReadFile(schemaFile)
	ts.Require().NoError(err)

	var schema map[string]any
	ts.Require().NoError(json.Unmarshal(schemaData, &schema))

	// Check that replicas from defaults is included
	props := schema["properties"].(map[string]any)
	replicas := props["replicas"].(map[string]any)
	ts.Equal("integer", replicas["type"])
	ts.Equal(float64(1), replicas["default"])

	// Check required fields - the implementation adds fields to required
	// even if they have defaults from the override file
	required, hasRequired := schema["required"].([]any)
	if hasRequired {
		// Both name and namespace are in values.yaml, so they're marked as required
		ts.ElementsMatch([]any{"namespace", "name"}, required)
	}
}

func (ts *ValetTestSuite) TestEndToEnd_ErrorHandling() {
	// Test various error scenarios
	tmpDir := ts.T().TempDir()
	
	oldDir, err := os.Getwd()
	ts.Require().NoError(err)
	ts.Require().NoError(os.Chdir(tmpDir))
	defer os.Chdir(oldDir)

	tests := []struct {
		name          string
		setup         func()
		args          []string
		expectedError string
	}{
		{
			name: "non-existent directory",
			setup: func() {},
			args: []string{"generate", "non-existent-dir"},
			expectedError: "no values.yaml or values.yml found",
		},
		{
			name: "invalid YAML syntax",
			setup: func() {
				invalidYAML := `
key: value
  bad: indentation
another: value
`
				ts.Require().NoError(os.WriteFile("values.yaml", []byte(invalidYAML), 0644))
			},
			args: []string{"generate", "."},
			expectedError: "yaml:",
		},
		{
			name: "empty file",
			setup: func() {
				ts.Require().NoError(os.WriteFile("values.yaml", []byte(""), 0644))
			},
			args: []string{"generate", "."},
			expectedError: "",  // Should succeed with empty schema
		},
	}

	for _, tt := range tests {
		ts.Run(tt.name, func() {
			tt.setup()
			
			rootCmd := cmd.NewRootCmd()
			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()
			
			if tt.expectedError != "" {
				ts.Error(err)
				ts.Contains(err.Error(), tt.expectedError)
			} else {
				ts.NoError(err)
			}
		})
	}
}

func (ts *ValetTestSuite) TestEndToEnd_ComplexDataTypes() {
	// Test handling of complex nested structures
	tmpDir := ts.T().TempDir()
	
	valuesContent := map[string]any{
		"mixedArray": []any{
			"string",
			123,
			true,
			map[string]any{"nested": "object"},
		},
		"nestedMaps": map[string]any{
			"level1": map[string]any{
				"level2": map[string]any{
					"level3": map[string]any{
						"value": "deep",
					},
				},
			},
		},
		"numbers": map[string]any{
			"integer": 42,
			"float": 3.14,
			"zero": 0,
			"negative": -10,
		},
		"booleans": map[string]any{
			"yes": true,
			"no": false,
		},
		"special": map[string]any{
			"null": nil,
			"empty": "",
			"spaces": "   ",
		},
	}

	valuesFile := filepath.Join(tmpDir, "complex.yaml")
	data, err := yaml.Marshal(valuesContent)
	ts.Require().NoError(err)
	ts.Require().NoError(os.WriteFile(valuesFile, data, 0644))

	oldDir, err := os.Getwd()
	ts.Require().NoError(err)
	ts.Require().NoError(os.Chdir(tmpDir))
	defer os.Chdir(oldDir)

	rootCmd := cmd.NewRootCmd()
	rootCmd.SetArgs([]string{"generate", tmpDir})
	ts.Require().NoError(rootCmd.Execute())

	schemaFile := filepath.Join(tmpDir, "complex.schema.json")
	schemaData, err := os.ReadFile(schemaFile)
	ts.Require().NoError(err)

	var schema map[string]any
	ts.Require().NoError(json.Unmarshal(schemaData, &schema))

	props := schema["properties"].(map[string]any)
	
	// Check mixed array handling
	mixedArray := props["mixedArray"].(map[string]any)
	ts.Equal("array", mixedArray["type"])
	items := mixedArray["items"].(map[string]any)
	ts.Contains(items, "oneOf") // Should use oneOf for mixed types
	
	// Check deep nesting
	nestedMaps := props["nestedMaps"].(map[string]any)
	level1 := nestedMaps["properties"].(map[string]any)["level1"].(map[string]any)
	level2 := level1["properties"].(map[string]any)["level2"].(map[string]any)
	level3 := level2["properties"].(map[string]any)["level3"].(map[string]any)
	value := level3["properties"].(map[string]any)["value"].(map[string]any)
	ts.Equal("string", value["type"])
	ts.Equal("deep", value["default"])
}