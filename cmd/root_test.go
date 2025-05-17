package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootCmd_DefaultContext(t *testing.T) {
	// Setup temporary directory with values.yaml
	tmp := t.TempDir()
	yaml := []byte("foo: bar\n")
	if err := os.WriteFile(filepath.Join(tmp, "values.yaml"), yaml, 0644); err != nil {
		t.Fatalf("failed to write values.yaml: %v", err)
	}
	// Change working directory to temp
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get cwd: %v", err)
	}
	defer os.Chdir(cwd)
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("cannot chdir: %v", err)
	}
	// Execute root command with no args
	// Reset global config
	cfg = nil
	cmd := NewRootCmd()
	cmd.SetArgs([]string{})
	// Execute command (output ignored)
	// cmd.SetOut/Stderr use default
	if err := cmd.Execute(); err != nil {
		t.Fatalf("root command failed: %v", err)
	}
	// Check file created
	schemaFile := filepath.Join(tmp, "values.schema.json")
	if _, err := os.Stat(schemaFile); err != nil {
		t.Errorf("expected schema file at %s, got error: %v", schemaFile, err)
	}
}

// TestInitializeConfig reads context from config-file and applies flag override
func TestInitializeConfig(t *testing.T) {
	tmp := t.TempDir()
	// Write config file with context 'foo'
	cfgPath := filepath.Join(tmp, "cfg.yaml")
	if err := os.WriteFile(cfgPath, []byte("context: foo\n"), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	// Case 1: config-file only
	cmd := NewRootCmd()
	cmd.PersistentFlags().Set("config-file", cfgPath)
	c, err := initializeConfig(cmd)
	if err != nil {
		t.Fatalf("initializeConfig failed: %v", err)
	}
	if c.Context != "foo" {
		t.Errorf("expected context 'foo', got '%s'", c.Context)
	}
	// Case 2: override via flag
	cmd2 := NewRootCmd()
	cmd2.PersistentFlags().Set("config-file", cfgPath)
	cmd2.PersistentFlags().Set("context", "bar")
	c2, err := initializeConfig(cmd2)
	if err != nil {
		t.Fatalf("initializeConfig failed: %v", err)
	}
	if c2.Context != "bar" {
		t.Errorf("expected context 'bar', got '%s'", c2.Context)
	}
}

// TestRootCmd_ConfigFile ensures context is read from default config file
func TestRootCmd_ConfigFile(t *testing.T) {
	tmp := t.TempDir()
	// Create a subdirectory with values.yaml
	sub := filepath.Join(tmp, "subdir")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	yaml := []byte("k: v\n")
	if err := os.WriteFile(filepath.Join(sub, "values.yaml"), yaml, 0644); err != nil {
		t.Fatalf("write values.yaml failed: %v", err)
	}
	// Create config file .valet.yaml in tmp
	cfgContent := []byte("context: subdir\n")
	if err := os.WriteFile(filepath.Join(tmp, ".valet.yaml"), cfgContent, 0644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}
	// Change to temp dir
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(tmp)
	// Reset global config
	cfg = nil
	cmd := NewRootCmd()
	// Specify config-file flag
	cmd.SetArgs([]string{"--config-file", ".valet.yaml"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	// Check schema file in subdir
	outPath := filepath.Join(sub, "values.schema.json")
	if _, err := os.Stat(outPath); err != nil {
		t.Errorf("expected schema at %s, got error: %v", outPath, err)
	}
}

// TestRootCmd_NoValues tests root command error when no values file present
func TestRootCmd_NoValues(t *testing.T) {
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(tmp)
	cfg = nil
	cmd := NewRootCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "no values.yaml or values.yml found in") {
		t.Errorf("expected missing values error, got %v", err)
	}
}
