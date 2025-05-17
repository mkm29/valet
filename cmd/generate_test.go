package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNewGenerateCmd(t *testing.T) {
	cmd := NewGenerateCmd()
	if cmd.Use != "generate <context-dir>" {
		t.Errorf("expected Use 'generate <context-dir>', got '%s'", cmd.Use)
	}
	if cmd.Short != "Generate JSON Schema from values.yaml" {
		t.Errorf("unexpected Short description: '%s'", cmd.Short)
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

// TestGenerateCmd_NoValues tests generate fails when no values file present
func TestGenerateCmd_NoValues(t *testing.T) {
   tmp := t.TempDir()
   cmd := NewGenerateCmd()
   // Capture error from Execute
   cmd.SetArgs([]string{tmp})
   err := cmd.Execute()
   if err == nil || !strings.Contains(err.Error(), "no values.yaml or values.yml found in") {
       t.Errorf("expected missing values error, got %v", err)
   }
}

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
	if msg != "Generated "+expectedMsg+" from values.yaml" {
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

// TestGenerateCommand_Execute runs the generate subcommand end-to-end
func TestGenerateCommand_Execute(t *testing.T) {
	tmp := t.TempDir()
	// Create values.yaml
	yaml := []byte("a: alpha\nb: beta\n")
	if err := os.WriteFile(filepath.Join(tmp, "values.yaml"), yaml, 0644); err != nil {
		t.Fatalf("write values.yaml failed: %v", err)
	}
	cmd := NewGenerateCmd()
	// Use absolute path to temp dir
	cmd.SetArgs([]string{tmp})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("GenerateCmd.Execute failed: %v", err)
	}
	// Check file exists at expected location
	outFile := filepath.Join(tmp, "values.schema.json")
	if _, err := os.Stat(outFile); err != nil {
		t.Errorf("expected schema file at %s, got error: %v", outFile, err)
	}
}

// TestGenerateCmd_MissingArg ensures subcommand errors on missing context arg
func TestGenerateCmd_MissingArg(t *testing.T) {
	cmd := NewGenerateCmd()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error when missing context argument")
	}
}

// TestGenerateCmd_Help ensures help text is shown without error
func TestGenerateCmd_Help(t *testing.T) {
	cmd := NewGenerateCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"-h"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("expected help to succeed, got %v", err)
	}
	if !strings.Contains(out.String(), "Generate JSON Schema") {
		t.Errorf("unexpected help output: %s", out.String())
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
	if msg != "Generated "+expectedMsg+" by merging over.yaml into values.yaml" {
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

// Test deepMerge merges maps and overrides scalars
func TestDeepMerge(t *testing.T) {
	a := map[string]any{
		"x": 1,
		"m": map[string]any{"a": 2, "b": 3},
		"s": "foo",
	}
	b := map[string]any{
		"x": 9,
		"m": map[string]any{"b": 7, "c": 8},
		"t": true,
	}
	merged := deepMerge(a, b)
	// Scalars: x overridden
	if merged["x"] != 9 {
		t.Errorf("expected x=9, got %v", merged["x"])
	}
	// Map m: nested merge
	mm, ok := merged["m"].(map[string]any)
	if !ok {
		t.Fatalf("expected m to be map, got %T", merged["m"])
	}
	expectM := map[string]any{"a": 2, "b": 7, "c": 8}
	if !reflect.DeepEqual(mm, expectM) {
		t.Errorf("expected m=%v, got %v", expectM, mm)
	}
	// s should remain
	if merged["s"] != "foo" {
		t.Errorf("expected s=foo, got %v", merged["s"])
	}
	// t should appear
	if merged["t"] != true {
		t.Errorf("expected t=true, got %v", merged["t"])
	}
}

// Test inferSchema simple types and defaults
func TestInferSchema_Primitives(t *testing.T) {
	// string
	sch := inferSchema("hello", "hello")
	if sch["type"] != "string" || sch["default"] != "hello" {
		t.Errorf("string schema incorrect: %v", sch)
	}
	// integer
	sch = inferSchema(42, 42)
	if sch["type"] != "integer" || sch["default"] != 42 {
		t.Errorf("integer schema incorrect: %v", sch)
	}
	// number
	sch = inferSchema(3.14, 3.14)
	if sch["type"] != "number" || sch["default"] != 3.14 {
		t.Errorf("number schema incorrect: %v", sch)
	}
	// boolean
	sch = inferSchema(true, true)
	if sch["type"] != "boolean" || sch["default"] != true {
		t.Errorf("boolean schema incorrect: %v", sch)
	}
}

// Test inferSchema for objects and required fields
func TestInferSchema_Object(t *testing.T) {
	val := map[string]any{"a": 1, "b": 0}
	def := map[string]any{"a": 1, "b": nil, "c": 3}
	sch := inferSchema(val, def)
	// required: only a (b default is nil), c not in val
	req, ok := sch["required"].([]string)
	if !ok || len(req) != 1 || req[0] != "a" {
		t.Errorf("required incorrect, got %v", sch["required"])
	}
}

// Test inferSchema for arrays
func TestInferSchema_Array(t *testing.T) {
	val := []any{1, 2}
	def := []any{0}
	sch := inferSchema(val, def)
	if sch["type"] != "array" {
		t.Errorf("expected array, got %v", sch["type"])
	}
	items, ok := sch["items"].(map[string]any)
	if !ok || items["default"] != 1 {
		t.Errorf("items schema incorrect: %v", sch["items"])
	}
}

// Test loadYAML missing and valid
func TestLoadYAML(t *testing.T) {
	tmp := t.TempDir()
	// missing file
	m, err := loadYAML(filepath.Join(tmp, "nofile.yaml"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
	// valid file
	content := "x: 10\ny: true"
	path := filepath.Join(tmp, "vals.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	m, err = loadYAML(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if m["x"] != 10 || m["y"] != true {
		t.Errorf("parsed map incorrect: %v", m)
	}
}
