package cmd

import (
   "encoding/json"
   "os"
   "path/filepath"
   "testing"
)

// Test basic schema generation without overrides
func TestGenerate_Simple(t *testing.T) {
   tmp := t.TempDir()
   // Create values.yaml
   yaml := []byte(
       "foo: bar\n" +
           "num: 42\n" +
           "flag: true\n",
   )
   if err := os.WriteFile(filepath.Join(tmp, "values.yaml"), yaml, 0644); err != nil {
       t.Fatalf("failed to write values.yaml: %v", err)
   }
   // Run Generate
   msg, err := Generate(tmp, "")
   if err != nil {
       t.Fatalf("Generate failed: %v", err)
   }
   // Expect message about generation
   expectedMsg := filepath.Join(tmp, "values.schema.json")
   if msg != "Generated " + expectedMsg + " from values.yaml" {
       t.Errorf("unexpected message: %s", msg)
   }
   // Read and unmarshal schema
   data, err := os.ReadFile(filepath.Join(tmp, "values.schema.json"))
   if err != nil {
       t.Fatalf("failed to read schema: %v", err)
   }
   var schema map[string]interface{}
   if err := json.Unmarshal(data, &schema); err != nil {
       t.Fatalf("invalid JSON schema: %v", err)
   }
   // Basic checks
   if schema["type"] != "object" {
       t.Errorf("expected type object, got %v", schema["type"])
   }
   props, ok := schema["properties"].(map[string]interface{})
   if !ok {
       t.Fatal("properties missing or wrong type")
   }
   // Check foo default
   foo, ok := props["foo"].(map[string]interface{})
   if !ok || foo["default"] != "bar" {
       t.Errorf("foo default incorrect: %v", foo)
   }
   // Check num default
   num, ok := props["num"].(map[string]interface{})
   if !ok {
       t.Error("num property missing")
   } else if num["default"] != float64(42) {
       t.Errorf("num default incorrect: %v", num["default"])
   }
   // Check flag default
   flagp, ok := props["flag"].(map[string]interface{})
   if !ok || flagp["default"] != true {
       t.Errorf("flag default incorrect: %v", flagp)
   }
}

// Test schema generation with overrides
func TestGenerate_Override(t *testing.T) {
   tmp := t.TempDir()
   // Create values.yaml
   yaml1 := []byte("a: 1\nes: test\n")
   if err := os.WriteFile(filepath.Join(tmp, "values.yaml"), yaml1, 0644); err != nil {
       t.Fatalf("failed to write values.yaml: %v", err)
   }
   // Create overrides.yaml
   yaml2 := []byte("a: 2\nb: new\n")
   if err := os.WriteFile(filepath.Join(tmp, "over.yaml"), yaml2, 0644); err != nil {
       t.Fatalf("failed to write overrides: %v", err)
   }
   msg, err := Generate(tmp, "over.yaml")
   if err != nil {
       t.Fatalf("Generate failed: %v", err)
   }
   expectedMsg := filepath.Join(tmp, "values.schema.json")
   if msg != "Generated " + expectedMsg + " by merging over.yaml into values.yaml" {
       t.Errorf("unexpected message: %s", msg)
   }
   // Read schema
   data, err := os.ReadFile(filepath.Join(tmp, "values.schema.json"))
   if err != nil {
       t.Fatalf("failed to read schema: %v", err)
   }
   var schema map[string]interface{}
   if err := json.Unmarshal(data, &schema); err != nil {
       t.Fatalf("invalid JSON schema: %v", err)
   }
   props := schema["properties"].(map[string]interface{})
   // a should be default 2
   a := props["a"].(map[string]interface{})
   if a["default"] != float64(2) {
       t.Errorf("override a default incorrect: %v", a["default"])
   }
   // b should appear
   b := props["b"].(map[string]interface{})
   if b["default"] != "new" {
       t.Errorf("override b default incorrect: %v", b["default"])
   }
}