package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
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

// TestMain_Error simulates missing values.yaml causing Execute error and exit
func TestMain_Error(t *testing.T) {
	tmp := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get cwd: %v", err)
	}
	defer os.Chdir(cwd)
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("cannot chdir: %v", err)
	}
	// No values.yaml present
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	// Use generate subcommand without context arg to force error
	os.Args = []string{"valet", "generate"}
	// Override exitFunc to panic with code
	oldExit := exitFunc
	defer func() { exitFunc = oldExit }()
	exitFunc = func(code int) { panic(code) }
	// Capture stderr
	r, w, _ := os.Pipe()
	oldStderr := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()
	var out bytes.Buffer
	// Run main in goroutine to catch panic
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				if code, ok := rec.(int); !ok || code != 1 {
					t.Errorf("expected exit code 1, got %v", rec)
				}
			} else {
				t.Error("expected exitFunc panic")
			}
			w.Close()
			io.Copy(&out, r)
		}()
		main()
	}()
	if !strings.Contains(out.String(), "Error:") {
		t.Errorf("expected error output, got %q", out.String())
	}
}
