package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/mkm29/valet/internal/config"
	"github.com/mkm29/valet/internal/telemetry"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRootCmd(t *testing.T) {
	cmd := NewRootCmd()

	assert.NotNil(t, cmd)
	assert.Equal(t, "valet", cmd.Use)
	assert.Equal(t, "JSON Schema Generator", cmd.Short)
	assert.True(t, cmd.SilenceUsage)
	assert.NotNil(t, cmd.PersistentPreRunE)
	assert.NotNil(t, cmd.PersistentPostRunE)
	assert.NotNil(t, cmd.RunE)

	// Check that subcommands are added
	var hasVersion, hasGenerate bool
	for _, subcmd := range cmd.Commands() {
		if subcmd.Use == "version" {
			hasVersion = true
		}
		if subcmd.Use == "generate <context-dir>" {
			hasGenerate = true
		}
	}
	assert.True(t, hasVersion, "version command should be added")
	assert.True(t, hasGenerate, "generate command should be added")

	// Check persistent flags
	flags := []string{
		"config-file", "context", "overrides", "output", "debug",
		"telemetry-enabled", "telemetry-exporter", "telemetry-endpoint",
		"telemetry-insecure", "telemetry-sample-rate",
	}

	for _, flag := range flags {
		f := cmd.PersistentFlags().Lookup(flag)
		assert.NotNil(t, f, "flag %s should exist", flag)
	}
}

func TestInitializeConfig(t *testing.T) {
	// Reset global config
	cfg = nil
	defer func() { cfg = nil }()

	tests := []struct {
		name           string
		setupFunc      func() (*cobra.Command, string)
		expectedConfig func(*config.Config)
		expectError    bool
	}{
		{
			name: "default config without file",
			setupFunc: func() (*cobra.Command, string) {
				cmd := &cobra.Command{}
				cmd.PersistentFlags().String("config-file", ".valet.yaml", "")
				cmd.PersistentFlags().String("context", ".", "")
				cmd.PersistentFlags().String("overrides", "", "")
				cmd.PersistentFlags().String("output", "values.schema.json", "")
				cmd.PersistentFlags().Bool("debug", false, "")
				cmd.PersistentFlags().Bool("telemetry-enabled", false, "")
				cmd.PersistentFlags().String("telemetry-exporter", "none", "")
				cmd.PersistentFlags().String("telemetry-endpoint", "localhost:4317", "")
				cmd.PersistentFlags().Bool("telemetry-insecure", false, "")
				cmd.PersistentFlags().Float64("telemetry-sample-rate", 1.0, "")
				return cmd, ""
			},
			expectedConfig: func(c *config.Config) {
				assert.Equal(t, ".", c.Context)
				assert.Equal(t, "", c.Overrides)
				assert.Equal(t, "", c.Output) // Not set unless flag is changed
				assert.False(t, c.Debug)
				assert.NotNil(t, c.Telemetry)
				assert.False(t, c.Telemetry.Enabled) // Default from NewTelemetryConfig
			},
			expectError: false,
		},
		{
			name: "config from file",
			setupFunc: func() (*cobra.Command, string) {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "test.yaml")
				content := `context: /test/path
overrides: test-overrides.yaml
output: test-output.json
debug: true
telemetry:
  enabled: true
  exporterType: stdout`
				err := os.WriteFile(configFile, []byte(content), 0644)
				require.NoError(t, err)

				cmd := &cobra.Command{}
				cmd.PersistentFlags().String("config-file", configFile, "")
				cmd.PersistentFlags().String("context", ".", "")
				cmd.PersistentFlags().String("overrides", "", "")
				cmd.PersistentFlags().String("output", "values.schema.json", "")
				cmd.PersistentFlags().Bool("debug", false, "")
				cmd.PersistentFlags().Bool("telemetry-enabled", false, "")
				cmd.PersistentFlags().String("telemetry-exporter", "none", "")
				cmd.PersistentFlags().String("telemetry-endpoint", "localhost:4317", "")
				cmd.PersistentFlags().Bool("telemetry-insecure", false, "")
				cmd.PersistentFlags().Float64("telemetry-sample-rate", 1.0, "")

				// Mark config-file as changed
				cmd.PersistentFlags().Set("config-file", configFile)

				return cmd, configFile
			},
			expectedConfig: func(c *config.Config) {
				assert.Equal(t, "/test/path", c.Context)
				assert.Equal(t, "test-overrides.yaml", c.Overrides)
				assert.Equal(t, "test-output.json", c.Output)
				assert.True(t, c.Debug)
				assert.True(t, c.Telemetry.Enabled)
				assert.Equal(t, "stdout", c.Telemetry.ExporterType)
			},
			expectError: false,
		},
		{
			name: "CLI flags override config file",
			setupFunc: func() (*cobra.Command, string) {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "test.yaml")
				content := `context: /file/path
debug: false`
				err := os.WriteFile(configFile, []byte(content), 0644)
				require.NoError(t, err)

				cmd := &cobra.Command{}
				cmd.PersistentFlags().String("config-file", configFile, "")
				cmd.PersistentFlags().String("context", "/cli/path", "")
				cmd.PersistentFlags().String("overrides", "", "")
				cmd.PersistentFlags().String("output", "values.schema.json", "")
				cmd.PersistentFlags().Bool("debug", true, "")
				cmd.PersistentFlags().Bool("telemetry-enabled", false, "")
				cmd.PersistentFlags().String("telemetry-exporter", "none", "")
				cmd.PersistentFlags().String("telemetry-endpoint", "localhost:4317", "")
				cmd.PersistentFlags().Bool("telemetry-insecure", false, "")
				cmd.PersistentFlags().Float64("telemetry-sample-rate", 1.0, "")

				// Mark flags as changed
				cmd.PersistentFlags().Set("config-file", configFile)
				cmd.PersistentFlags().Set("context", "/cli/path")
				cmd.PersistentFlags().Set("debug", "true")

				return cmd, configFile
			},
			expectedConfig: func(c *config.Config) {
				assert.Equal(t, "/cli/path", c.Context)
				assert.True(t, c.Debug)
			},
			expectError: false,
		},
		{
			name: "telemetry flags",
			setupFunc: func() (*cobra.Command, string) {
				cmd := &cobra.Command{}
				cmd.PersistentFlags().String("config-file", ".valet.yaml", "")
				cmd.PersistentFlags().String("context", ".", "")
				cmd.PersistentFlags().String("overrides", "", "")
				cmd.PersistentFlags().String("output", "values.schema.json", "")
				cmd.PersistentFlags().Bool("debug", false, "")
				cmd.PersistentFlags().Bool("telemetry-enabled", true, "")
				cmd.PersistentFlags().String("telemetry-exporter", "otlp", "")
				cmd.PersistentFlags().String("telemetry-endpoint", "custom:4317", "")
				cmd.PersistentFlags().Bool("telemetry-insecure", true, "")
				cmd.PersistentFlags().Float64("telemetry-sample-rate", 0.5, "")

				// Mark telemetry flags as changed
				cmd.PersistentFlags().Set("telemetry-enabled", "true")
				cmd.PersistentFlags().Set("telemetry-exporter", "otlp")
				cmd.PersistentFlags().Set("telemetry-endpoint", "custom:4317")
				cmd.PersistentFlags().Set("telemetry-insecure", "true")
				cmd.PersistentFlags().Set("telemetry-sample-rate", "0.5")

				return cmd, ""
			},
			expectedConfig: func(c *config.Config) {
				assert.True(t, c.Telemetry.Enabled)
				assert.Equal(t, "otlp", c.Telemetry.ExporterType)
				assert.Equal(t, "custom:4317", c.Telemetry.OTLPEndpoint)
				assert.True(t, c.Telemetry.Insecure)
				assert.Equal(t, 0.5, c.Telemetry.SampleRate)
			},
			expectError: false,
		},
		{
			name: "invalid config file",
			setupFunc: func() (*cobra.Command, string) {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "invalid.yaml")
				content := `invalid yaml content
  - bad indentation`
				err := os.WriteFile(configFile, []byte(content), 0644)
				require.NoError(t, err)

				cmd := &cobra.Command{}
				cmd.PersistentFlags().String("config-file", configFile, "")
				cmd.PersistentFlags().String("context", ".", "")
				cmd.PersistentFlags().String("overrides", "", "")
				cmd.PersistentFlags().String("output", "values.schema.json", "")
				cmd.PersistentFlags().Bool("debug", false, "")
				cmd.PersistentFlags().Bool("telemetry-enabled", false, "")
				cmd.PersistentFlags().String("telemetry-exporter", "none", "")
				cmd.PersistentFlags().String("telemetry-endpoint", "localhost:4317", "")
				cmd.PersistentFlags().Bool("telemetry-insecure", false, "")
				cmd.PersistentFlags().Float64("telemetry-sample-rate", 1.0, "")

				// Mark config-file as changed
				cmd.PersistentFlags().Set("config-file", configFile)

				return cmd, configFile
			},
			expectedConfig: nil,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, _ := tt.setupFunc()

			cfg, err := initializeConfig(cmd)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
				if tt.expectedConfig != nil {
					tt.expectedConfig(cfg)
				}
			}
		})
	}
}

func TestRootCmdExecution(t *testing.T) {
	// Reset globals
	cfg = nil
	tel = nil
	defer func() {
		cfg = nil
		tel = nil
	}()

	tests := []struct {
		name        string
		args        []string
		setupFunc   func() string
		expectError bool
		checkOutput func(t *testing.T, output string)
	}{
		{
			name:        "help when no context",
			args:        []string{},
			setupFunc:   func() string { return "" },
			expectError: true, // Will error because no values.yaml in current dir
			checkOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "no values.yaml")
			},
		},
		{
			name: "generate with context arg",
			args: []string{"generate", "testdir"},
			setupFunc: func() string {
				tmpDir := t.TempDir()
				testDir := filepath.Join(tmpDir, "testdir")
				err := os.Mkdir(testDir, 0755)
				require.NoError(t, err)

				valuesContent := `key: value`
				err = os.WriteFile(filepath.Join(testDir, "values.yaml"), []byte(valuesContent), 0644)
				require.NoError(t, err)

				return tmpDir
			},
			expectError: false,
			checkOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "Generated")
			},
		},
		{
			name: "generate with missing values.yaml",
			args: []string{"generate", "testdir"},
			setupFunc: func() string {
				tmpDir := t.TempDir()
				testDir := filepath.Join(tmpDir, "testdir")
				err := os.Mkdir(testDir, 0755)
				require.NoError(t, err)
				return tmpDir
			},
			expectError: true,
			checkOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "no values.yaml")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := tt.setupFunc()

			cmd := NewRootCmd()
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			// Adjust args to use full path if baseDir is provided
			args := tt.args
			if baseDir != "" && len(args) > 1 && args[0] == "generate" {
				args[1] = filepath.Join(baseDir, args[1])
			}
			cmd.SetArgs(args)

			err := cmd.Execute()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.checkOutput != nil {
				tt.checkOutput(t, buf.String())
			}
		})
	}
}

func TestGetTelemetry(t *testing.T) {
	// Reset global telemetry
	tel = nil
	defer func() { tel = nil }()

	// Initially should be nil
	assert.Nil(t, GetTelemetry())

	// Create a mock telemetry instance
	mockTel := &telemetry.Telemetry{}
	tel = mockTel

	// Should return the set instance
	assert.Equal(t, mockTel, GetTelemetry())
}

func TestPersistentPreRunE(t *testing.T) {
	// Reset globals
	cfg = nil
	tel = nil
	defer func() {
		cfg = nil
		tel = nil
	}()

	cmd := NewRootCmd()

	// Test with telemetry disabled (default)
	err := cmd.PersistentPreRunE(cmd, []string{})
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	// Even with telemetry disabled, tel gets initialized but it's a no-op implementation
	// when built with notelemetry build tag

	// Test with telemetry enabled
	cfg = nil
	tel = nil
	cmd.SetArgs([]string{"--telemetry-enabled", "--telemetry-exporter", "none"})
	cmd.ParseFlags([]string{"--telemetry-enabled", "--telemetry-exporter", "none"})

	err = cmd.PersistentPreRunE(cmd, []string{})
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.NotNil(t, tel) // Should be initialized with telemetry enabled
}

func TestPersistentPostRunE(t *testing.T) {
	// Reset globals
	cfg = nil
	tel = nil
	defer func() {
		cfg = nil
		tel = nil
	}()

	cmd := NewRootCmd()

	// Test with no telemetry
	err := cmd.PersistentPostRunE(cmd, []string{})
	assert.NoError(t, err)
}
