package utils

import (
	"fmt"
	"os"
)

// BuildNestedDefaults builds defaults for nested map values
func BuildNestedDefaults(mapVal map[string]any) map[string]any {
	nestedDefaults := make(map[string]any)
	for k, v := range mapVal {
		if v != nil {
			nestedDefaults[k] = v
		}
	}
	return nestedDefaults
}

// BuildObjectDefaults builds the default values for an object schema
func BuildObjectDefaults(v map[string]any) map[string]any {
	defaults := make(map[string]any, len(v))
	for key, val := range v {
		// Skip null values in defaults
		if val == nil {
			continue
		}

		// Process map values correctly for defaults
		mapVal, isMap := val.(map[string]any)
		if isMap {
			// For maps, process the defaults recursively
			nestedDefaults := BuildNestedDefaults(mapVal)
			if len(nestedDefaults) > 0 {
				defaults[key] = nestedDefaults
			}
		} else {
			// For non-maps, include the value directly
			defaults[key] = val
		}
	}
	return defaults
}

// IsNullValue checks if a value represents null
func IsNullValue(val any) bool {
	if val == nil {
		return true
	}

	if strVal, ok := val.(string); ok {
		return strVal == "null" || strVal == ""
	}

	return false
}

// IsDisabledComponent checks if a component has enabled=false
func IsDisabledComponent(val any) bool {
	component, ok := val.(map[string]any)
	if !ok {
		return false
	}

	if enabled, exists := component["enabled"]; exists {
		if enabledBool, isBool := enabled.(bool); isBool && !enabledBool {
			return true
		}
	}

	return false
}

// IsChildOfDisabledComponent checks if the parent object has enabled=false
func IsChildOfDisabledComponent(parentObject map[string]any) bool {
	if parent, ok := parentObject["enabled"]; ok {
		if parentEnabled, ok := parent.(bool); ok && !parentEnabled {
			return true
		}
	}
	return false
}

// CountSchemaFields counts the number of fields in a schema recursively
func CountSchemaFields(schema map[string]any) int {
	count := 0
	if props, ok := schema["properties"].(map[string]any); ok {
		count += len(props)
		for _, prop := range props {
			if propMap, ok := prop.(map[string]any); ok {
				count += CountSchemaFields(propMap)
			}
		}
	}
	return count
}

// InferBooleanSchema processes boolean types
func InferBooleanSchema(v bool) map[string]any {
	return map[string]any{
		"type":    "boolean",
		"default": v,
	}
}

// InferIntegerSchema processes integer types
func InferIntegerSchema(v any) map[string]any {
	return map[string]any{
		"type":    "integer",
		"default": v,
	}
}

// InferNumberSchema processes float64 types, converting to integer if appropriate
func InferNumberSchema(v float64) map[string]any {
	if float64(int64(v)) == v {
		return map[string]any{
			"type":    "integer",
			"default": int64(v),
		}
	}
	return map[string]any{
		"type":    "number",
		"default": v,
	}
}

// InferStringSchema processes string types, handling null strings specially
func InferStringSchema(v string) map[string]any {
	if v == "null" || v == "<nil>" || v == "" {
		typeArray := []string{"string", "null"}
		return map[string]any{
			"type":    typeArray,
			"default": nil,
		}
	}
	return map[string]any{
		"type":    "string",
		"default": v,
	}
}

// InferNullSchema returns a schema for null values
func InferNullSchema() map[string]any {
	typeArray := []string{"string", "null"}
	return map[string]any{
		"type":    typeArray,
		"default": nil,
	}
}

// InferReflectedFloatSchema handles reflected float types
func InferReflectedFloatSchema(floatVal float64) map[string]any {
	// Check if it's actually an integer
	if floatVal == float64(int64(floatVal)) {
		return map[string]any{
			"type":    "integer",
			"default": int64(floatVal),
		}
	}
	return map[string]any{
		"type":    "number",
		"default": floatVal,
	}
}

// GetContextDirectory extracts the context directory from arguments
func GetContextDirectory(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	// Get the current working directory
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current working directory:", err)
		return ""
	}
	return dir
}

// InferArraySchema processes array types and generates array schema
// The inferItemSchema parameter should be a function that infers schema for array items
func InferArraySchema(v []any, defaultVal any, inferItemSchema func(any, any) map[string]any) map[string]any {
	var defItem any
	if defArr, ok := defaultVal.([]any); ok && len(defArr) > 0 {
		defItem = defArr[0]
	}

	var itemsSchema map[string]any
	if len(v) > 0 {
		itemsSchema = inferItemSchema(v[0], defItem)
	} else {
		itemsSchema = map[string]any{}
	}

	return map[string]any{
		"type":    "array",
		"items":   itemsSchema,
		"default": v,
	}
}
