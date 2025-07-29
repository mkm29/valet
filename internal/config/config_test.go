package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTelemetryConfig(t *testing.T) {
	cfg := NewTelemetryConfig()

	assert.NotNil(t, cfg)
	assert.False(t, cfg.Enabled)
	assert.Equal(t, "valet", cfg.ServiceName)
	assert.Equal(t, "0.1.0", cfg.ServiceVersion)
	assert.Equal(t, "none", cfg.ExporterType)
	assert.Equal(t, "localhost:4317", cfg.OTLPEndpoint)
	assert.True(t, cfg.Insecure)
	assert.Equal(t, 1.0, cfg.SampleRate)
	assert.NotNil(t, cfg.Headers)
	assert.Empty(t, cfg.Headers)
}

func TestTelemetryConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *TelemetryConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid disabled config",
			config: &TelemetryConfig{
				Enabled:      false,
				ExporterType: "none",
			},
			wantErr: false,
		},
		{
			name: "valid enabled config with none exporter",
			config: &TelemetryConfig{
				Enabled:      true,
				ExporterType: "none",
			},
			wantErr: false,
		},
		{
			name: "valid enabled config with stdout exporter",
			config: &TelemetryConfig{
				Enabled:      true,
				ExporterType: "stdout",
			},
			wantErr: false,
		},
		{
			name: "valid enabled config with otlp exporter",
			config: &TelemetryConfig{
				Enabled:      true,
				ExporterType: "otlp",
				OTLPEndpoint: "localhost:4317",
			},
			wantErr: false,
		},
		{
			name: "invalid exporter type",
			config: &TelemetryConfig{
				Enabled:      true,
				ExporterType: "invalid",
			},
			wantErr: true,
			errMsg:  "invalid exporter type",
		},
		{
			name: "otlp exporter without endpoint",
			config: &TelemetryConfig{
				Enabled:      true,
				ExporterType: "otlp",
				OTLPEndpoint: "",
			},
			wantErr: false, // Empty endpoint is allowed, uses default
			errMsg:  "",
		},
		{
			name: "invalid sample rate - negative",
			config: &TelemetryConfig{
				Enabled:      true,
				ExporterType: "none",
				SampleRate:   -0.1,
			},
			wantErr: true,
			errMsg:  "sample rate must be between 0.0 and 1.0",
		},
		{
			name: "invalid sample rate - too high",
			config: &TelemetryConfig{
				Enabled:      true,
				ExporterType: "none",
				SampleRate:   1.5,
			},
			wantErr: true,
			errMsg:  "sample rate must be between 0.0 and 1.0",
		},
		{
			name: "valid sample rate - zero",
			config: &TelemetryConfig{
				Enabled:      true,
				ExporterType: "none",
				SampleRate:   0.0,
			},
			wantErr: false,
		},
		{
			name: "valid sample rate - one",
			config: &TelemetryConfig{
				Enabled:      true,
				ExporterType: "none",
				SampleRate:   1.0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name           string
		setup          func() (string, func())
		envVars        map[string]string
		expectedConfig *Config
		expectError    bool
	}{
		{
			name: "default config when no file exists",
			setup: func() (string, func()) {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "nonexistent.yaml"), func() {}
			},
			expectedConfig: &Config{
				Context:   "", // Default is empty
				Overrides: "",
				Output:    "", // Default is empty
				Debug:     false,
				Telemetry: NewTelemetryConfig(),
			},
			expectError: false,
		},
		{
			name: "load from config file",
			setup: func() (string, func()) {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "config.yaml")

				content := `
context: /path/to/context
overrides: overrides.yaml
output: custom.schema.json
debug: true
telemetry:
  enabled: true
  serviceName: custom-service
  serviceVersion: 2.0.0
  exporterType: stdout
  sampleRate: 0.5
`
				err := os.WriteFile(configFile, []byte(content), 0644)
				require.NoError(t, err)

				return configFile, func() {}
			},
			expectedConfig: &Config{
				Context:   "/path/to/context",
				Overrides: "overrides.yaml",
				Output:    "custom.schema.json",
				Debug:     true,
				Telemetry: &TelemetryConfig{
					Enabled:        true,
					ServiceName:    "custom-service",
					ServiceVersion: "2.0.0",
					ExporterType:   "stdout",
					OTLPEndpoint:   "localhost:4317",
					Insecure:       true, // Default from NewTelemetryConfig is true
					SampleRate:     0.5,
					Headers:        map[string]string{},
				},
			},
			expectError: false,
		},
		{
			name: "environment variables override config file",
			setup: func() (string, func()) {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "config.yaml")

				content := `
context: /file/context
overrides: file-overrides.yaml
output: file.schema.json
debug: false
`
				err := os.WriteFile(configFile, []byte(content), 0644)
				require.NoError(t, err)

				return configFile, func() {}
			},
			envVars: map[string]string{
				"VALET_CONTEXT":   "/env/context",
				"VALET_OVERRIDES": "env-overrides.yaml",
				"VALET_OUTPUT":    "env.schema.json",
				"VALET_DEBUG":     "true",
			},
			expectedConfig: &Config{
				Context:   "/file/context", // LoadConfig doesn't read env vars
				Overrides: "file-overrides.yaml",
				Output:    "file.schema.json", // From file
				Debug:     false,
				Telemetry: NewTelemetryConfig(),
			},
			expectError: false,
		},
		{
			name: "partial config file",
			setup: func() (string, func()) {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "config.yaml")

				content := `
context: /partial/context
debug: true
`
				err := os.WriteFile(configFile, []byte(content), 0644)
				require.NoError(t, err)

				return configFile, func() {}
			},
			expectedConfig: &Config{
				Context:   "/partial/context",
				Overrides: "",
				Output:    "", // LoadConfig doesn't set defaults, only reads from file
				Debug:     true,
				Telemetry: NewTelemetryConfig(),
			},
			expectError: false,
		},
		{
			name: "invalid yaml in config file",
			setup: func() (string, func()) {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "config.yaml")

				content := `
context: /path/to/context
  - invalid yaml structure
    bad: indentation
    - another bad list
debug: true:
`
				err := os.WriteFile(configFile, []byte(content), 0644)
				require.NoError(t, err)

				return configFile, func() {}
			},
			expectError: true,
		},
		{
			name: "config file with permission error",
			setup: func() (string, func()) {
				if os.Getuid() == 0 {
					t.Skip("Cannot test permission errors as root")
				}

				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "config.yaml")

				content := `context: /test`
				err := os.WriteFile(configFile, []byte(content), 0644)
				require.NoError(t, err)

				// Make file unreadable
				err = os.Chmod(configFile, 0000)
				require.NoError(t, err)

				return configFile, func() {
					os.Chmod(configFile, 0644)
				}
			},
			expectError: true,
		},
		{
			name: "telemetry config from environment",
			setup: func() (string, func()) {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "nonexistent.yaml"), func() {}
			},
			envVars: map[string]string{
				"VALET_TELEMETRY":                 "true",
				"VALET_TELEMETRY_EXPORTER":        "otlp",
				"VALET_TELEMETRY_ENDPOINT":        "custom:4317",
				"VALET_TELEMETRY_INSECURE":        "true",
				"VALET_TELEMETRY_SAMPLE_RATE":     "0.25",
				"VALET_TELEMETRY_SERVICE_NAME":    "env-service",
				"VALET_TELEMETRY_SERVICE_VERSION": "3.0.0",
			},
			expectedConfig: &Config{
				Context:   "", // LoadConfig doesn't read env vars
				Overrides: "",
				Output:    "",
				Debug:     false,
				Telemetry: NewTelemetryConfig(), // LoadConfig doesn't read env vars
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			configFile, cleanup := tt.setup()
			defer cleanup()

			// Set environment variables
			for k, v := range tt.envVars {
				oldVal := os.Getenv(k)
				os.Setenv(k, v)
				defer os.Setenv(k, oldVal)
			}

			// Load config
			cfg, err := LoadConfig(configFile)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, cfg)

				// Compare configs
				assert.Equal(t, tt.expectedConfig.Context, cfg.Context)
				assert.Equal(t, tt.expectedConfig.Overrides, cfg.Overrides)
				assert.Equal(t, tt.expectedConfig.Output, cfg.Output)
				assert.Equal(t, tt.expectedConfig.Debug, cfg.Debug)

				if tt.expectedConfig.Telemetry != nil {
					assert.Equal(t, tt.expectedConfig.Telemetry.Enabled, cfg.Telemetry.Enabled)
					assert.Equal(t, tt.expectedConfig.Telemetry.ServiceName, cfg.Telemetry.ServiceName)
					assert.Equal(t, tt.expectedConfig.Telemetry.ServiceVersion, cfg.Telemetry.ServiceVersion)
					assert.Equal(t, tt.expectedConfig.Telemetry.ExporterType, cfg.Telemetry.ExporterType)
					assert.Equal(t, tt.expectedConfig.Telemetry.OTLPEndpoint, cfg.Telemetry.OTLPEndpoint)
					assert.Equal(t, tt.expectedConfig.Telemetry.Insecure, cfg.Telemetry.Insecure)
					assert.Equal(t, tt.expectedConfig.Telemetry.SampleRate, cfg.Telemetry.SampleRate)
				}
			}
		})
	}
}

func TestConfigPrecedence(t *testing.T) {
	// Create a config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	content := `
context: file-context
overrides: file-overrides.yaml
output: file-output.json
debug: false
telemetry:
  enabled: false
`
	err := os.WriteFile(configFile, []byte(content), 0644)
	require.NoError(t, err)

	// Set some environment variables (note: LoadConfig doesn't read these)
	os.Setenv("VALET_CONTEXT", "env-context")
	os.Setenv("VALET_DEBUG", "true")
	defer os.Unsetenv("VALET_CONTEXT")
	defer os.Unsetenv("VALET_DEBUG")

	// Load config
	cfg, err := LoadConfig(configFile)
	require.NoError(t, err)

	// LoadConfig only reads from file, not environment
	assert.Equal(t, "file-context", cfg.Context)
	assert.False(t, cfg.Debug)

	// File values should be used
	assert.Equal(t, "file-overrides.yaml", cfg.Overrides)
	assert.Equal(t, "file-output.json", cfg.Output)
}
