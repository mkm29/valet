package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Test that main() generates a schema file using default context
func TestMain_Generate(t *testing.T) {
	tmp := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get cwd: %v", err)
	}
	defer os.Chdir(cwd)
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("cannot chdir: %v", err)
	}
	// Create a values.yaml for generation
	yaml := []byte("a: b\n")
	if err := os.WriteFile("values.yaml", yaml, 0644); err != nil {
		t.Fatalf("failed to write values.yaml: %v", err)
	}
	// Set args to enable default execution
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"valet"}
	// Run main
	main()
	// Check that schema file was created
	outFile := filepath.Join(tmp, "values.schema.json")
	if _, err := os.Stat(outFile); err != nil {
		t.Errorf("expected schema file at %s, got error: %v", outFile, err)
	}
}
