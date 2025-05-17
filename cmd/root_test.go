package cmd

import (
   "bytes"
   "os"
   "path/filepath"
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
   cmd := NewRootCmd()
   cmd.SetArgs([]string{})
   var out bytes.Buffer
   cmd.SetOut(&out)
   cmd.SetErr(&out)
   if err := cmd.Execute(); err != nil {
       t.Fatalf("root command failed: %v", err)
   }
   // Check file created
   schemaFile := filepath.Join(tmp, "values.schema.json")
   if _, err := os.Stat(schemaFile); err != nil {
       t.Errorf("expected schema file at %s, got error: %v", schemaFile, err)
   }
}