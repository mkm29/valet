package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mkm29/valet/internal/config"
)

func TestLoadConfig_NoFile(t *testing.T) {
	// No config file present
	cfg, err := config.LoadConfig("nonexistent.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Context != "" {
		t.Errorf("expected empty Context, got %q", cfg.Context)
	}
	if cfg.Overrides != "" {
		t.Errorf("expected empty Overrides, got %q", cfg.Overrides)
	}
	if cfg.Output != "" {
		t.Errorf("expected empty Output, got %q", cfg.Output)
	}
	if cfg.Debug {
		t.Error("expected Debug=false by default")
	}
}

func TestLoadConfig_FromFile(t *testing.T) {
	tmp := t.TempDir()
	// Create a config file
	cfgFile := filepath.Join(tmp, "valet.yaml")
	data := []byte(
		"context: foo_dir\n" +
			"overrides: override.yaml\n" +
			"output: out.json\n" +
			"debug: true\n",
	)
	if err := os.WriteFile(cfgFile, data, 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Context != "foo_dir" {
		t.Errorf("expected Context=foo_dir, got %q", cfg.Context)
	}
	if cfg.Overrides != "override.yaml" {
		t.Errorf("expected Overrides=override.yaml, got %q", cfg.Overrides)
	}
	if cfg.Output != "out.json" {
		t.Errorf("expected Output=out.json, got %q", cfg.Output)
	}
	if !cfg.Debug {
		t.Error("expected Debug=true from config file")
	}
}

// TestLoadConfig_BadYAML ensures parse errors are returned
func TestLoadConfig_BadYAML(t *testing.T) {
	tmp := t.TempDir()
	cfgFile := filepath.Join(tmp, "bad.yaml")
	// Write invalid YAML
	if err := os.WriteFile(cfgFile, []byte("not: [bad_yaml"), 0644); err != nil {
		t.Fatalf("failed to write bad YAML: %v", err)
	}
	_, err := config.LoadConfig(cfgFile)
	if err == nil || !strings.Contains(err.Error(), "failed to parse config") {
		t.Errorf("expected parse error, got %v", err)
	}
}

// TestLoadConfig_EnvVarsAndDefaults tests loading configs with all options set
func TestLoadConfig_EnvVarsAndDefaults(t *testing.T) {
	// Create config with all options
	content := `
debug: true
context: /config/context
overrides: config-values.yaml
output: config-schema.json
`
	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	
	// Load config
	cfg, err := config.LoadConfig(tmpFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	
	// Verify values
	if !cfg.Debug {
		t.Error("Debug should be true")
	}
	if cfg.Context != "/config/context" {
		t.Errorf("Context incorrect, got %s", cfg.Context)
	}
	if cfg.Overrides != "config-values.yaml" {
		t.Errorf("Overrides incorrect, got %s", cfg.Overrides)
	}
	if cfg.Output != "config-schema.json" {
		t.Errorf("Output incorrect, got %s", cfg.Output)
	}
}

// TestLoadConfig_Partial tests loading configs with partial options
func TestLoadConfig_Partial(t *testing.T) {
	// Create config with partial options
	content := `
debug: true
# Intentionally omitting other fields
`
	tmpFile := filepath.Join(t.TempDir(), "config-partial.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	
	// Load config
	cfg, err := config.LoadConfig(tmpFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	
	// Verify values
	if !cfg.Debug {
		t.Error("Debug should be true")
	}
	if cfg.Context != "" {
		t.Errorf("Context should be empty, got %s", cfg.Context)
	}
	if cfg.Overrides != "" {
		t.Errorf("Overrides should be empty, got %s", cfg.Overrides)
	}
	if cfg.Output != "" {
		t.Errorf("Output should be empty, got %s", cfg.Output)
	}
}

// TestLoadConfig_FilePermissionError tests LoadConfig with unreadable file
func TestLoadConfig_FilePermissionError(t *testing.T) {
	// Skip on Windows where chmod doesn't work the same
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping on Windows")
	}
	
	// Create temp file
	tmpFile := filepath.Join(t.TempDir(), "config-perm.yaml")
	if err := os.WriteFile(tmpFile, []byte("debug: true"), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	
	// Make file unreadable (this won't work on Windows)
	if err := os.Chmod(tmpFile, 0000); err != nil {
		t.Fatalf("failed to chmod: %v", err)
	}
	
	// Load config - should return error
	_, err := config.LoadConfig(tmpFile)
	if err == nil {
		t.Error("expected error for unreadable file")
	}
	
	// Fix permissions for cleanup
	os.Chmod(tmpFile, 0644)
}

// TestLoadConfig_ComplexYAML tests loading configs with complex YAML structures
func TestLoadConfig_ComplexYAML(t *testing.T) {
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
	tmpFile := filepath.Join(t.TempDir(), "config-complex.yaml") 
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	
	// Load config
	cfg, err := config.LoadConfig(tmpFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	
	// Verify critical values
	if !cfg.Debug {
		t.Error("Debug should be true")
	}
	if cfg.Context != "/complex/context" {
		t.Errorf("Context incorrect, got %s", cfg.Context)
	}
}
