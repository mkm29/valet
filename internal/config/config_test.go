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
