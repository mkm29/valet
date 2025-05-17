package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDeepMerge_Shallow(t *testing.T) {
	a := map[string]any{"a": 1, "b": 2}
	b := map[string]any{"b": 3, "c": 4}
	got := deepMerge(a, b)
	want := map[string]any{"a": 1, "b": 3, "c": 4}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("deepMerge(shallow) = %v, want %v", got, want)
	}
}

// Test version flag in main, expecting exit code 0 and version info.
func TestMain_VersionFlag(t *testing.T) {
	// reset flags before parsing in main
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	origArgs := os.Args
	origExit := exit
	defer func() {
		os.Args = origArgs
		exit = origExit
	}()
	exit = func(code int) { panic(code) }
	reader, writer, _ := os.Pipe()
	origStdout := os.Stdout
	os.Stdout = writer
	defer func() { os.Stdout = origStdout }()
	os.Args = []string{"valet", "-version"}
	var code int
	defer func() {
		if r := recover(); r != nil {
			if c, ok := r.(int); ok {
				code = c
			} else {
				t.Fatalf("unexpected panic: %v", r)
			}
		} else {
			t.Fatalf("expected exit panic")
		}
		writer.Close()
		b, _ := io.ReadAll(reader)
		out := string(b)
		if code != 0 {
			t.Errorf("exit code = %d, want 0", code)
		}
		if !strings.HasPrefix(out, "github.com/mkm29/valet@") {
			t.Errorf("version output = %q, want prefix %q", out, "github.com/mkm29/valet@")
		}
	}()
	main()
}

// Test missing args in main, expecting exit code 1 and usage printed.
func TestMain_MissingArgs(t *testing.T) {
	// reset flags before parsing in main
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	origArgs := os.Args
	origExit := exit
	defer func() {
		os.Args = origArgs
		exit = origExit
	}()
	exit = func(code int) { panic(code) }
	reader, writer, _ := os.Pipe()
	origStderr := os.Stderr
	os.Stderr = writer
	defer func() { os.Stderr = origStderr }()
	os.Args = []string{"valet"}
	var code int
	defer func() {
		if r := recover(); r != nil {
			if c, ok := r.(int); ok {
				code = c
			} else {
				t.Fatalf("unexpected panic: %v", r)
			}
		} else {
			t.Fatalf("expected exit panic")
		}
		writer.Close()
		b, _ := io.ReadAll(reader)
		out := string(b)
		if code != 1 {
			t.Errorf("exit code = %d, want 1", code)
		}
		if !strings.Contains(out, "Usage: valet [flags] <context-dir>") {
			t.Errorf("usage output = %q, missing usage prefix", out)
		}
	}()
	main()
}

func TestDeepMerge_Nested(t *testing.T) {
	a := map[string]any{"n": map[string]any{"x": 1, "y": 1}}
	b := map[string]any{"n": map[string]any{"y": 2, "z": 3}}
	got := deepMerge(a, b)
	want := map[string]any{"n": map[string]any{"x": 1, "y": 2, "z": 3}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("deepMerge(nested) = %v, want %v", got, want)
	}
}

func TestInferSchema_Primitives(t *testing.T) {
	tests := []struct {
		name     string
		val      any
		def      any
		wantType string
		wantDef  any
	}{
		{"bool", true, nil, "boolean", true},
		{"int", 42, nil, "integer", 42},
		{"floatInt", 5.0, nil, "integer", int64(5)},
		{"floatNum", 5.5, nil, "number", 5.5},
		{"string", "s", nil, "string", "s"},
		{"default", struct{ A int }{A: 1}, nil, "string", fmt.Sprintf("%v", struct{ A int }{A: 1})},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			schema := inferSchema(tc.val, tc.def)
			if schemaType, _ := schema["type"].(string); schemaType != tc.wantType {
				t.Errorf("inferSchema[%s] type = %s, want %s", tc.name, schemaType, tc.wantType)
			}
			if def := schema["default"]; !reflect.DeepEqual(def, tc.wantDef) {
				t.Errorf("inferSchema[%s] default = %v, want %v", tc.name, def, tc.wantDef)
			}
		})
	}
}

func TestInferSchema_ArrayAndObject(t *testing.T) {
	// Empty array
	schema := inferSchema([]any{}, []any{})
	if schema["type"] != "array" {
		t.Errorf("empty array type = %v, want array", schema["type"])
	}
	// Array with object
	valArr := []any{map[string]any{"n": 1}}
	defArr := []any{map[string]any{"n": 0}}
	schema2 := inferSchema(valArr, defArr)
	items, ok := schema2["items"].(map[string]any)
	if !ok {
		t.Fatalf("items not map, got %T", schema2["items"])
	}
	props, _ := items["properties"].(map[string]any)
	defVal := props["n"].(map[string]any)["default"]
	if !reflect.DeepEqual(defVal, 1) {
		t.Errorf("array item default = %v, want 1", defVal)
	}
	// Object with required keys
	obj := map[string]any{"a": 1, "b": 2}
	defObj := map[string]any{"a": 0}
	schemaObj := inferSchema(obj, defObj)
	propsObj, _ := schemaObj["properties"].(map[string]any)
	if _, ok := propsObj["a"]; !ok {
		t.Errorf("properties missing 'a'")
	}
	req, _ := schemaObj["required"].([]string)
	if len(req) != 1 || req[0] != "a" {
		t.Errorf("required = %v, want [a]", req)
	}
}

func TestInferSchema_Object_NoRequired(t *testing.T) {
	obj := map[string]any{"a": 1}
	defObj := map[string]any{}
	schema := inferSchema(obj, defObj)
	if _, ok := schema["required"]; ok {
		t.Errorf("expected no required, got %v", schema["required"])
	}
}

func TestInferSchema_EmptyArrayDefault(t *testing.T) {
	schema := inferSchema([]any{}, []any{})
	def, ok := schema["default"].([]any)
	if !ok {
		t.Fatalf("default not []any, got %T", schema["default"])
	}
	if len(def) != 0 {
		t.Errorf("default slice length = %d, want 0", len(def))
	}
}

func TestInferSchema_EmptyObjectDefault(t *testing.T) {
	obj := map[string]any{}
	schema := inferSchema(obj, obj)
	def, ok := schema["default"].(map[string]any)
	if !ok {
		t.Fatalf("default not map[string]any, got %T", schema["default"])
	}
	if len(def) != 0 {
		t.Errorf("default map length = %d, want 0", len(def))
	}
}

func TestLoadYAML_Missing(t *testing.T) {
	dir := t.TempDir()
	m, err := loadYAML(filepath.Join(dir, "noexist.yaml"))
	if err != nil {
		t.Errorf("loadYAML missing returned error: %v", err)
	}
	if len(m) != 0 {
		t.Errorf("loadYAML missing = %v, want empty map", m)
	}
}

func TestLoadYAML_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "empty.yaml")
	os.WriteFile(p, []byte(""), 0644)
	m, err := loadYAML(p)
	if err != nil {
		t.Fatalf("loadYAML empty error: %v", err)
	}
	if len(m) != 0 {
		t.Errorf("loadYAML empty = %v, want empty map", m)
	}
}

func TestLoadYAML_Invalid(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.yaml")
	os.WriteFile(p, []byte("foo: [bar"), 0644)
	if _, err := loadYAML(p); err == nil {
		t.Errorf("loadYAML invalid did not return error")
	}
}

func TestGenerate_NoOverrides(t *testing.T) {
	dir := t.TempDir()
	// Write values.yaml
	val := "key: val"
	os.WriteFile(filepath.Join(dir, "values.yaml"), []byte(val), 0644)
	msg, err := Generate(dir, "")
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if !strings.HasSuffix(msg, "from values.yaml") {
		t.Errorf("Generate msg = %q, want suffix 'from values.yaml'", msg)
	}
	// Check output file
	out := filepath.Join(dir, "values.schema.json")
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	var s map[string]any
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	props, _ := s["properties"].(map[string]any)
	d := props["key"].(map[string]any)["default"]
	if d != "val" {
		t.Errorf("default for key = %v, want 'val'", d)
	}
}

func TestGenerate_WithOverrides(t *testing.T) {
	dir := t.TempDir()
	// Write base and override
	os.WriteFile(filepath.Join(dir, "values.yaml"), []byte("a: 1\nb: 2"), 0644)
	os.WriteFile(filepath.Join(dir, "override.yaml"), []byte("b: 3\nc: 4"), 0644)
	msg, err := Generate(dir, "override.yaml")
	if err != nil {
		t.Fatalf("Generate override error: %v", err)
	}
	if !strings.Contains(msg, "by merging override.yaml into values.yaml") {
		t.Errorf("Generate msg = %q, want merge message", msg)
	}
	// Verify merged defaults
	data, _ := os.ReadFile(filepath.Join(dir, "values.schema.json"))
	var s map[string]any
	json.Unmarshal(data, &s)
	props := s["properties"].(map[string]any)
	ba := props["a"].(map[string]any)["default"]
	bb := props["b"].(map[string]any)["default"]
	bc := props["c"].(map[string]any)["default"]
	if ba != float64(1) || bb != float64(3) || bc != float64(4) {
		t.Errorf("merged defaults = %v,%v,%v, want 1,3,4", ba, bb, bc)
	}
}

func TestGenerate_InvalidOverride(t *testing.T) {
	dir := t.TempDir()
	// base ok, override invalid
	os.WriteFile(filepath.Join(dir, "values.yaml"), []byte("k: v"), 0644)
	os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte("foo: [bar"), 0644)
	if _, err := Generate(dir, "bad.yaml"); err == nil {
		t.Errorf("Generate with invalid override did not return error")
	}
}

// Missing override file should still succeed (empty merge)
func TestGenerate_MissingOverrideFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "values.yaml"), []byte("x: 9"), 0644)
	msg, err := Generate(dir, "no.yaml")
	if err != nil {
		t.Fatalf("Generate missing override error: %v", err)
	}
	if !strings.Contains(msg, "by merging no.yaml into values.yaml") {
		t.Errorf("Generate msg = %q, want merge message", msg)
	}
	// Check output default
	data, _ := os.ReadFile(filepath.Join(dir, "values.schema.json"))
	var s map[string]any
	json.Unmarshal(data, &s)
	def := s["properties"].(map[string]any)["x"].(map[string]any)["default"]
	if !reflect.DeepEqual(def, float64(9)) && !reflect.DeepEqual(def, 9) {
		t.Errorf("default for x = %v, want 9", def)
	}
}

// Invalid base values.yaml should return error
func TestGenerate_InvalidValuesYAML(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "values.yaml"), []byte("foo: [bar"), 0644)
	if _, err := Generate(dir, ""); err == nil {
		t.Errorf("Generate with invalid base YAML did not return error")
	}
}

func TestLoadYAML_ReadError(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "file")
	os.WriteFile(p, []byte("data"), 0644)
	// Attempt to read subpath under file, should error
	child := filepath.Join(p, "child.yaml")
	if _, err := loadYAML(child); err == nil {
		t.Errorf("loadYAML read error did not return error")
	}
}

// Write error when output directory is not writable
func TestGenerate_WriteError(t *testing.T) {
	parent := t.TempDir()
	ctx := filepath.Join(parent, "ctx")
	// create directory and write values.yaml before restricting permissions
	if err := os.Mkdir(ctx, 0755); err != nil {
		t.Fatalf("mkdir ctx: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ctx, "values.yaml"), []byte("k: v"), 0644); err != nil {
		t.Fatalf("write values.yaml: %v", err)
	}
	// now make directory non-writable
	if err := os.Chmod(ctx, 0500); err != nil {
		t.Fatalf("chmod ctx: %v", err)
	}
	// ensure we restore permissions so TempDir cleanup succeeds
	defer func() {
		os.Chmod(ctx, 0755)
	}()
	_, err := Generate(ctx, "")
	if err == nil {
		t.Fatalf("expected write error, got nil")
	}
	if !strings.HasPrefix(err.Error(), "error writing") {
		t.Errorf("error = %v, want prefix 'error writing'", err)
	}
}

// Test main() end-to-end flow with valid args
func TestMain_Flow(t *testing.T) {
	dir := t.TempDir()
	// write values.yaml
	content := "foo: bar\nnum: 7"
	if err := os.WriteFile(filepath.Join(dir, "values.yaml"), []byte(content), 0644); err != nil {
		t.Fatalf("write values.yaml: %v", err)
	}
	// capture stdout
	// reset flags before parsing in main
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"valet", dir}
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	// run main
	main()
	w.Close()
	os.Stdout = old
	outBytes, _ := io.ReadAll(r)
	out := string(outBytes)
	expected := fmt.Sprintf("Generated %s from values.yaml", filepath.Join(dir, "values.schema.json"))
	if !strings.Contains(out, expected) {
		t.Errorf("main output = %q, want %q", out, expected)
	}
	// verify output file
	if _, err := os.Stat(filepath.Join(dir, "values.schema.json")); err != nil {
		t.Errorf("output file not created: %v", err)
	}
}
