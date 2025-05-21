package cmd

import (
	"reflect"
	"testing"
)

// TestInferSchema_FullCoverage adds comprehensive tests for the inferSchema function
func TestInferSchema_FullCoverage(t *testing.T) {
	// Test different numeric types
	intTests := []struct {
		name  string
		value any
		want  string
	}{
		{"int", 42, "integer"},
		{"int64", int64(42), "integer"},
		{"float64_integer", float64(42), "integer"},
		{"float64_decimal", 3.14, "number"},
	}
	
	for _, tt := range intTests {
		t.Run(tt.name, func(t *testing.T) {
			schema := inferSchema(tt.value, tt.value)
			if schema["type"] != tt.want {
				t.Errorf("type = %v, want %v", schema["type"], tt.want)
			}
		})
	}
	
	// Test string types and special cases
	stringTests := []struct {
		name     string
		value    string
		wantType any
	}{
		{"string", "hello", "string"},
		{"empty_string", "", []string{"string", "null"}},
		{"null_string", "null", []string{"string", "null"}},
		{"nil_string", "<nil>", []string{"string", "null"}},
	}
	
	for _, tt := range stringTests {
		t.Run(tt.name, func(t *testing.T) {
			schema := inferSchema(tt.value, tt.value)
			if !reflect.DeepEqual(schema["type"], tt.wantType) {
				t.Errorf("type = %v, want %v", schema["type"], tt.wantType)
			}
		})
	}
	
	// Test boolean values
	boolSchema := inferSchema(true, true)
	if boolSchema["type"] != "boolean" || boolSchema["default"] != true {
		t.Errorf("boolean schema incorrect: %v", boolSchema)
	}
	
	// Test complex objects with inheritance
	parentVal := map[string]any{
		"name": "parent",
		"children": []any{
			map[string]any{"id": 1, "name": "child1"},
		},
	}
	
	schema := inferSchema(parentVal, parentVal)
	if schema["type"] != "object" {
		t.Errorf("expected object type, got %v", schema["type"])
	}
	
	props := schema["properties"].(map[string]any)
	if props["name"] == nil || props["children"] == nil {
		t.Errorf("missing properties in schema: %v", props)
	}
	
	childrenProp := props["children"].(map[string]any)
	if childrenProp["type"] != "array" {
		t.Errorf("expected array type for children, got %v", childrenProp["type"])
	}
	
	// Test reflection-based type discovery
	reflectionTests := []struct {
		name  string
		value any
		want  string
	}{
		{"reflect_map", map[string]string{"a": "b"}, "object"},
		{"reflect_slice", []string{"a", "b"}, "array"},
		{"reflect_bool", true, "boolean"},
		{"reflect_int", int(42), "integer"},
		{"reflect_int8", int8(42), "integer"},
		{"reflect_int16", int16(42), "integer"},
		{"reflect_int32", int32(42), "integer"},
		{"reflect_int64", int64(42), "integer"},
		{"reflect_uint", uint(42), "integer"},
		{"reflect_uint8", uint8(42), "integer"},
		{"reflect_uint16", uint16(42), "integer"},
		{"reflect_uint32", uint32(42), "integer"},
		{"reflect_uint64", uint64(42), "integer"},
		{"reflect_float32", float32(3.14), "number"},
		{"reflect_float64", float64(3.14), "number"},
		{"reflect_float32_int", float32(42.0), "integer"},
	}
	
	for _, tt := range reflectionTests {
		t.Run(tt.name, func(t *testing.T) {
			schema := inferSchema(tt.value, tt.value)
			if schema["type"] != tt.want {
				t.Errorf("type = %v, want %v", schema["type"], tt.want)
			}
		})
	}
	
	// Test nested objects
	nestedVal := map[string]any{
		"level1": map[string]any{
			"level2": map[string]any{
				"level3": "value",
			},
		},
	}
	
	schema = inferSchema(nestedVal, nestedVal)
	props = schema["properties"].(map[string]any)
	level1 := props["level1"].(map[string]any)
	level1Props := level1["properties"].(map[string]any)
	level2 := level1Props["level2"].(map[string]any)
	level2Props := level2["properties"].(map[string]any)
	level3 := level2Props["level3"].(map[string]any)
	
	if level3["type"] != "string" || level3["default"] != "value" {
		t.Errorf("incorrect nested schema: %v", level3)
	}
	
	// Test map with different default value
	val := map[string]any{"a": 1, "b": 2}
	defVal := map[string]any{"a": 5, "c": 3}
	schema = inferSchema(val, defVal)
	props = schema["properties"].(map[string]any)
	if props["c"] != nil {
		t.Error("property 'c' should not be included")
	}
	
	// Check required field
	required, ok := schema["required"].([]string)
	if !ok || len(required) == 0 || required[0] != "a" {
		t.Errorf("required fields incorrect: %v", schema["required"])
	}
}

// TestIsEmptyValue_AdvancedCases adds more tests for isEmptyValue
func TestIsEmptyValue_AdvancedCases(t *testing.T) {
	// Test pointer types
	var nilPtr *string
	if !isEmptyValue(nilPtr) {
		t.Error("nil pointer should be empty")
	}
	
	strVal := "hello"
	strPtr := &strVal
	if isEmptyValue(strPtr) {
		t.Error("non-nil pointer should not be empty")
	}
	
	// Test map types
	emptyMap := make(map[string]interface{})
	if !isEmptyValue(emptyMap) {
		t.Error("empty map should be empty")
	}
	
	// Test slice types
	var nilSlice []int
	if !isEmptyValue(nilSlice) {
		t.Error("nil slice should be empty")
	}
	
	emptySlice := make([]int, 0)
	if !isEmptyValue(emptySlice) {
		t.Error("empty slice should be empty")
	}
	
	// Test array type
	var emptyArray [0]int
	if !isEmptyValue(emptyArray) {
		t.Error("empty array should be empty")
	}
	
	// Test struct type - not directly supported by isEmptyValue but should not crash
	type emptyStruct struct{}
	if isEmptyValue(emptyStruct{}) {
		// This is actually expected to return false since we don't have specific handling for it
		t.Error("empty struct unexpectedly detected as empty")
	}
}

// TestProcessProperties_EdgeCases adds edge case tests for processProperties
func TestProcessProperties_EdgeCases(t *testing.T) {
	// Test with invalid properties
	schema := map[string]any{
		"type": "object",
	}
	processProperties(schema, nil)
	// Should not crash
	
	// Test with properties but no required
	schema = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"foo": map[string]any{"type": "string"},
		},
	}
	processProperties(schema, map[string]any{"foo": "bar"})
	// Should not crash or modify schema
	
	// Test with empty array of required - the function might delete the empty required field
	// This is more of a behavior verification than a correctness check
	schema = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"foo": map[string]any{"type": "string"},
		},
		"required": []string{},
	}
	processProperties(schema, map[string]any{"foo": "bar"})
	// Should not crash, and either remove required or keep it empty
	
	// Test with all required items removed
	schema = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"foo": map[string]any{"type": "string"},
		},
		"required": []string{"foo"},
	}
	defaults := map[string]any{
		"foo": "", // Empty string
	}
	processProperties(schema, defaults)
	// Should remove required completely
	if _, exists := schema["required"]; exists {
		t.Errorf("required should be removed when all items are filtered out")
	}
}

// TestConvertToStringKeyMap_Advanced tests the convertToStringKeyMap function with more complex inputs
func TestConvertToStringKeyMap_Advanced(t *testing.T) {
	// Test with deeply nested maps
	input := map[interface{}]interface{}{
		"key1": map[interface{}]interface{}{
			"nested1": map[interface{}]interface{}{
				"deep1": "value1",
				1:       "numeric key",
			},
			"array": []interface{}{
				map[interface{}]interface{}{
					"item1": "value",
				},
				"string item",
				42,
			},
		},
		2: "top-level numeric key",
	}
	
	result := convertToStringKeyMap(input)
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", result)
	}
	
	// Check top level
	if resultMap["key1"] == nil || resultMap["2"] != "top-level numeric key" {
		t.Errorf("top level conversion failed, got %v", resultMap)
	}
	
	// Check first level of nesting
	nestedMap, ok := resultMap["key1"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected nested map, got %T", resultMap["key1"])
	}
	
	// Check second level of nesting
	deepMap, ok := nestedMap["nested1"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected deep map, got %T", nestedMap["nested1"])
	}
	
	if deepMap["deep1"] != "value1" || deepMap["1"] != "numeric key" {
		t.Errorf("deep map conversion failed, got %v", deepMap)
	}
	
	// Check array conversion
	arr, ok := nestedMap["array"].([]interface{})
	if !ok || len(arr) != 3 {
		t.Fatalf("expected array with 3 items, got %v", nestedMap["array"])
	}
	
	// Check array item that's a map
	arrMap, ok := arr[0].(map[string]interface{})
	if !ok || arrMap["item1"] != "value" {
		t.Errorf("array map conversion failed, got %v", arr[0])
	}
	
	// Check non-map array items remain unchanged
	if arr[1] != "string item" || arr[2] != 42 {
		t.Errorf("primitive array items changed, got %v and %v", arr[1], arr[2])
	}
}