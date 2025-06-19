package tests

import (
	"os"
	"path/filepath"

	"github.com/mkm29/valet/internal/config"
)

func (ts *ValetTestSuite) TestLoadConfig_NoFile() {
	// No config file present
	cfg, err := config.LoadConfig("nonexistent.yaml")
	ts.NoError(err, "unexpected error")
	ts.Empty(cfg.Context, "expected empty Context")
	ts.Empty(cfg.Overrides, "expected empty Overrides")
	ts.Empty(cfg.Output, "expected empty Output")
	ts.False(cfg.Debug, "expected Debug=false by default")
}

func (ts *ValetTestSuite) TestLoadConfig_FromFile() {
	tmp := ts.T().TempDir()
	// Create a config file
	cfgFile := filepath.Join(tmp, "valet.yaml")
	data := []byte(
		"context: foo_dir\n" +
			"overrides: override.yaml\n" +
			"output: out.json\n" +
			"debug: true\n",
	)
	err := os.WriteFile(cfgFile, data, 0644)
	ts.Require().NoError(err, "failed to write config file")

	cfg, err := config.LoadConfig(cfgFile)
	ts.NoError(err, "unexpected error")
	ts.Equal("foo_dir", cfg.Context, "expected Context=foo_dir")
	ts.Equal("override.yaml", cfg.Overrides, "expected Overrides=override.yaml")
	ts.Equal("out.json", cfg.Output, "expected Output=out.json")
	ts.True(cfg.Debug, "expected Debug=true from config file")
}

// TestLoadConfig_BadYAML ensures parse errors are returned
func (ts *ValetTestSuite) TestLoadConfig_BadYAML() {
	tmp := ts.T().TempDir()
	cfgFile := filepath.Join(tmp, "bad.yaml")
	// Write invalid YAML
	err := os.WriteFile(cfgFile, []byte("not: [bad_yaml"), 0644)
	ts.Require().NoError(err, "failed to write bad YAML")

	_, err = config.LoadConfig(cfgFile)
	ts.Error(err)
	ts.Contains(err.Error(), "failed to parse config", "expected parse error")
}

// TestLoadConfig_EnvVarsAndDefaults tests loading configs with all options set
func (ts *ValetTestSuite) TestLoadConfig_EnvVarsAndDefaults() {
	// Create config with all options
	content := `
debug: true
context: /config/context
overrides: config-values.yaml
output: config-schema.json
`
	tmpFile := filepath.Join(ts.T().TempDir(), "config.yaml")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	ts.Require().NoError(err, "failed to write config")

	// Load config
	cfg, err := config.LoadConfig(tmpFile)
	ts.NoError(err, "LoadConfig failed")

	// Verify values
	ts.True(cfg.Debug, "Debug should be true")
	ts.Equal("/config/context", cfg.Context, "Context incorrect")
	ts.Equal("config-values.yaml", cfg.Overrides, "Overrides incorrect")
	ts.Equal("config-schema.json", cfg.Output, "Output incorrect")
}

// TestLoadConfig_Partial tests loading configs with partial options
func (ts *ValetTestSuite) TestLoadConfig_Partial() {
	// Create config with partial options
	content := `
debug: true
# Intentionally omitting other fields
`
	tmpFile := filepath.Join(ts.T().TempDir(), "config-partial.yaml")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	ts.Require().NoError(err, "failed to write config")

	// Load config
	cfg, err := config.LoadConfig(tmpFile)
	ts.NoError(err, "LoadConfig failed")

	// Verify values
	ts.True(cfg.Debug, "Debug should be true")
	ts.Empty(cfg.Context, "Context should be empty")
	ts.Empty(cfg.Overrides, "Overrides should be empty")
	ts.Empty(cfg.Output, "Output should be empty")
}

// TestLoadConfig_FilePermissionError tests LoadConfig with unreadable file
func (ts *ValetTestSuite) TestLoadConfig_FilePermissionError() {
	// Skip on Windows where chmod doesn't work the same
	if os.Getenv("GOOS") == "windows" {
		ts.T().Skip("Skipping on Windows")
	}

	// Create temp file
	tmpFile := filepath.Join(ts.T().TempDir(), "config-perm.yaml")
	err := os.WriteFile(tmpFile, []byte("debug: true"), 0644)
	ts.Require().NoError(err, "failed to write config")

	// Make file unreadable (this won't work on Windows)
	err = os.Chmod(tmpFile, 0000)
	ts.Require().NoError(err, "failed to chmod")

	// Load config - should return error
	_, err = config.LoadConfig(tmpFile)
	ts.Error(err, "expected error for unreadable file")

	// Fix permissions for cleanup
	os.Chmod(tmpFile, 0644)
}

// TestLoadConfig_ComplexYAML tests loading configs with complex YAML structures
func (ts *ValetTestSuite) TestLoadConfig_ComplexYAML() {
	// Create config with complex YAML (we'll ignore most of these fields)
	content := `
debug: true
context: /complex/context
# The following fields are ignored but should parse correctly
extra:
  nested:
    value: something
  list:
    - item1
    - item2
mappings:
  key1: value1
  key2: value2
`
	tmpFile := filepath.Join(ts.T().TempDir(), "config-complex.yaml")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	ts.Require().NoError(err, "failed to write config")

	// Load config
	cfg, err := config.LoadConfig(tmpFile)
	ts.NoError(err, "LoadConfig failed")

	// Verify critical values
	ts.True(cfg.Debug, "Debug should be true")
	ts.Equal("/complex/context", cfg.Context, "Context incorrect")
}