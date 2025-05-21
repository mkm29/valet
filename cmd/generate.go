package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// generate subcommand

// deepMerge merges b into a (recursively for nested maps) and returns a new map.
func deepMerge(a, b map[string]any) map[string]any {
	out := make(map[string]any, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, vb := range b {
		if va, ok := out[k]; ok {
			ma, maOK := va.(map[string]any)
			mb, mbOK := vb.(map[string]any)
			if maOK && mbOK {
				out[k] = deepMerge(ma, mb)
				continue
			}
		}
		out[k] = vb
	}
	return out
}

// inferSchema builds a JSON‐Schema fragment for val, using defaultVal
// to determine which object keys are "required".
func inferSchema(val, defaultVal any) map[string]any {
	switch v := val.(type) {
	case map[string]any:
		defMap, _ := defaultVal.(map[string]any)
		props := make(map[string]any, len(v))
		for key, sub := range v {
			// Ensure we pass the correct default value for the subfield
			var defSubVal any
			if defMap != nil {
				defSubVal = defMap[key]
			}
			props[key] = inferSchema(sub, defSubVal)
		}

		// Build default object with actual values from the YAML
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
				nestedDefaults := make(map[string]any)
				for k, v := range mapVal {
					if v != nil {
						nestedDefaults[k] = v
					}
				}
				if len(nestedDefaults) > 0 {
					defaults[key] = nestedDefaults
				}
			} else {
				// For non-maps, include the value directly
				defaults[key] = val
			}
		}

		schema := map[string]any{
			"type":       "object",
			"properties": props,
			"default":    defaults,
		}

		var required []string
		for k, vDefault := range defMap {
			// Only add to required if key exists, default is NOT nil and NOT null (in Go or YAML)
			if _, exists := v[k]; exists {
				isNil := vDefault == nil
				isNullString := false
				switch vDefault := vDefault.(type) {
				case string:
					// YAML nulls sometimes decode as string "null" or as empty string
					isNullString = vDefault == "null" || vDefault == ""
				}

				// Special handling for components that can be enabled/disabled
				if !isNil && !isNullString {
					// Check for empty values (empty strings, arrays, objects)
					// Skip fields with empty default values using the helper function
					if isEmptyValue(vDefault) {
						isDebug := cfg != nil && cfg.Debug
						if isDebug {
							fmt.Printf("Skipping field because it has an empty default value of type %T\n", vDefault)
						}
						continue
					}

					// Check if this is a component that can be enabled/disabled
					component, isComponent := v[k].(map[string]any)
					if isComponent {
						// Check if this component has an 'enabled' field
						if enabled, exists := component["enabled"]; exists {
							// Only add fields as required if the component is enabled by default
							if enabledBool, isBool := enabled.(bool); isBool && !enabledBool {
								// Component is disabled by default, don't add to required
								continue
							}
						}

						// Check if this is a property of a parent component with an enabled field
						if parent, ok := v["enabled"]; ok {
							if parentEnabled, ok := parent.(bool); ok && !parentEnabled {
								// Parent component is disabled, don't mark this as required
								continue
							}
						}
					}

					// Normal fields and enabled components are added as required
					required = append(required, k)
				}
			}
		}
		if len(required) > 0 {
			schema["required"] = required
		}
		return schema

	case []any:
		var defItem any
		if defArr, ok := defaultVal.([]any); ok && len(defArr) > 0 {
			defItem = defArr[0]
		}
		var itemsSchema map[string]any
		if len(v) > 0 {
			itemsSchema = inferSchema(v[0], defItem)
		} else {
			itemsSchema = map[string]any{}
		}
		return map[string]any{
			"type":    "array",
			"items":   itemsSchema,
			"default": v,
		}

	case bool:
		return map[string]any{
			"type":    "boolean",
			"default": v,
		}

	case int, int64:
		return map[string]any{
			"type":    "integer",
			"default": v,
		}

	case float64:
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

	case string:
		// Handle null strings specially
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

	default:
		// Handle unknown types more intelligently using reflection
		rv := reflect.ValueOf(v)

		// Handle nil values
		if !rv.IsValid() || (rv.Kind() == reflect.Ptr && rv.IsNil()) {
			typeArray := []string{"string", "null"}
			return map[string]any{
				"type":    typeArray,
				"default": nil,
			}
		}

		// Get the underlying value if it's a pointer
		if rv.Kind() == reflect.Ptr {
			rv = rv.Elem()
		}

		if rv.Kind() == reflect.Map {
			// Convert maps to proper JSON objects
			defMap := make(map[string]any)
			for _, k := range rv.MapKeys() {
				if k.Kind() == reflect.String {
					mv := rv.MapIndex(k).Interface()
					// Skip nil values
					if mv != nil {
						defMap[k.String()] = mv
					}
				}
			}

			// Recursively process properties
			props := make(map[string]any)
			for k, sub := range defMap {
				// Get corresponding default value if available
				var defVal any
				if defaultMap, ok := defaultVal.(map[string]any); ok {
					defVal = defaultMap[k]
				}
				props[k] = inferSchema(sub, defVal)
			}

			return map[string]any{
				"type":       "object",
				"properties": props,
				"default":    defMap,
			}
		} else if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			// Convert slices to proper arrays
			var items []any
			for i := 0; i < rv.Len(); i++ {
				item := rv.Index(i).Interface()
				if item != nil {
					items = append(items, item)
				}
			}

			var itemsSchema map[string]any
			if len(items) > 0 {
				// Use first item to infer schema
				var defItem any
				if defArr, ok := defaultVal.([]any); ok && len(defArr) > 0 {
					defItem = defArr[0]
				}
				itemsSchema = inferSchema(items[0], defItem)
			} else {
				itemsSchema = map[string]any{}
			}

			return map[string]any{
				"type":    "array",
				"items":   itemsSchema,
				"default": items,
			}
		} else if rv.Kind() == reflect.Bool {
			return map[string]any{
				"type":    "boolean",
				"default": rv.Bool(),
			}
		} else if rv.Kind() == reflect.Int || rv.Kind() == reflect.Int8 ||
			rv.Kind() == reflect.Int16 || rv.Kind() == reflect.Int32 ||
			rv.Kind() == reflect.Int64 {
			return map[string]any{
				"type":    "integer",
				"default": rv.Int(),
			}
		} else if rv.Kind() == reflect.Uint || rv.Kind() == reflect.Uint8 ||
			rv.Kind() == reflect.Uint16 || rv.Kind() == reflect.Uint32 ||
			rv.Kind() == reflect.Uint64 {
			return map[string]any{
				"type":    "integer",
				"default": rv.Uint(),
			}
		} else if rv.Kind() == reflect.Float32 || rv.Kind() == reflect.Float64 {
			floatVal := rv.Float()
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
		} else if rv.Kind() == reflect.String {
			strVal := rv.String()
			// Handle "null" string representations
			if strVal == "null" || strVal == "<nil>" {
				typeArray := []string{"string", "null"}
				return map[string]any{
					"type":    typeArray,
					"default": nil,
				}
			}
			return map[string]any{
				"type":    "string",
				"default": strVal,
			}
		} else {
			// Fall back to string representation for other types
			return map[string]any{
				"type":    "string",
				"default": fmt.Sprintf("%v", v),
			}
		}
	}
}

// convertToStringKeyMap recursively converts map[interface{}]interface{} to map[string]interface{}
func convertToStringKeyMap(m interface{}) interface{} {
	switch x := m.(type) {
	case map[interface{}]interface{}:
		result := make(map[string]interface{})
		for k, v := range x {
			result[fmt.Sprintf("%v", k)] = convertToStringKeyMap(v)
		}
		return result
	case []interface{}:
		for i, v := range x {
			x[i] = convertToStringKeyMap(v)
		}
	}
	return m
}

// loadYAML reads a YAML file into map[string]any (empty if missing)
func loadYAML(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	var m map[interface{}]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m == nil {
		return map[string]any{}, nil
	}

	// Convert to map[string]any
	result := convertToStringKeyMap(m).(map[string]interface{})
	return result, nil
}

// Generate a JSON Schema for the values.yaml in ctx directory,
// optionally merging an overrides YAML file relative to ctx.
// It writes the schema to values.schema.json and returns a status message.
func Generate(ctx, overridesFlag string) (string, error) {
	// Locate values file (values.yaml or values.yml)
	valuesPath := filepath.Join(ctx, "values.yaml")
	if _, err := os.Stat(valuesPath); os.IsNotExist(err) {
		alt := filepath.Join(ctx, "values.yml")
		if _, err2 := os.Stat(alt); os.IsNotExist(err2) {
			return "", fmt.Errorf("no values.yaml or values.yml found in %s", ctx)
		}
		valuesPath = alt
	}
	var overridesPath string
	if overridesFlag != "" {
		overridesPath = filepath.Join(ctx, overridesFlag)
	}
	yaml1, err := loadYAML(valuesPath)
	if err != nil {
		return "", fmt.Errorf("error loading %s: %w", valuesPath, err)
	}

	// Log some of the top-level default values to help with debugging
	// Use safe debugging to handle cases when cfg is nil (testing environment)
	isDebug := cfg != nil && cfg.Debug
	if isDebug {
		fmt.Println("Original YAML values from", valuesPath)
		
		// Print debug info for top-level components in a generic way
		fmt.Println("Components with 'enabled' field:")
		enabledComponentCount := 0
		for _, compVal := range yaml1 {
			if compMap, isMap := compVal.(map[string]any); isMap {
				// Log components with enabled field
				if enabled, hasEnabled := compMap["enabled"]; hasEnabled {
					enabledComponentCount++
					if enabledBool, isBool := enabled.(bool); isBool {
						fmt.Printf("  Component %d: %v\n", enabledComponentCount, enabledBool)
					}
				}
			}
		}
	}

	var merged map[string]any
	if overridesPath != "" {
		yaml2, err := loadYAML(overridesPath)
		if err != nil {
			return "", fmt.Errorf("error loading %s: %w", overridesPath, err)
		}
		merged = deepMerge(yaml1, yaml2)
	} else {
		merged = yaml1
	}
	schema := inferSchema(merged, yaml1)
	schema["$schema"] = "http://json-schema.org/schema#"

	// Post-process the schema to ensure no empty fields are in the required lists
	cleanupRequiredFields(schema, yaml1)

	outPath := filepath.Join(ctx, "values.schema.json")
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error marshaling JSON: %w", err)
	}
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return "", fmt.Errorf("error writing %s: %w", outPath, err)
	}
	if overridesPath != "" {
		return fmt.Sprintf("Generated %s by merging %s into values.yaml", outPath, overridesFlag), nil
	}
	return fmt.Sprintf("Generated %s from values.yaml", outPath), nil
}

// isEmptyValue checks if a value represents an empty value (empty string, array, map)
func isEmptyValue(val any) bool {
	if val == nil {
		return true
	}

	switch v := val.(type) {
	case string:
		return v == ""
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	case map[interface{}]interface{}:
		return len(v) == 0
	}

	// Use reflection for other types
	rv := reflect.ValueOf(val)

	// Handle nil values
	if !rv.IsValid() || (rv.Kind() == reflect.Ptr && rv.IsNil()) {
		return true
	}

	// Get the underlying value if it's a pointer
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	if rv.Kind() == reflect.Map {
		return rv.Len() == 0
	} else if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		return rv.Len() == 0
	}

	return false
}

// cleanupRequiredFields processes the generated schema to ensure no empty values are in required lists
func cleanupRequiredFields(schema map[string]any, defaults map[string]any) {
	// Process the top-level schema
	processProperties(schema, defaults)
}

// processProperties handles each property in the schema, recursing into nested objects
func processProperties(schema map[string]any, defaults map[string]any) {
	// Check if this is an object with properties
	properties, hasProps := schema["properties"].(map[string]any)
	if !hasProps {
		return
	}

	// Get the required fields list
	required, hasRequired := schema["required"].([]string)
	if !hasRequired {
		return
	}

	// Check if this object itself has an 'enabled' key that is false
	if enabled, hasEnabled := defaults["enabled"]; hasEnabled {
		if enabledBool, isBool := enabled.(bool); isBool && !enabledBool {
			isDebug := cfg != nil && cfg.Debug
			if isDebug {
				fmt.Printf("Post-processing: Removing required fields from component because it has enabled=false\n")
			}
			delete(schema, "required")
			return
		}
	}

	// Go through each property and check if it has an empty default value
	var newRequired []string
	for _, fieldName := range required {
		// Get the default value for this field
		defVal, hasDef := defaults[fieldName]

		// If the default value is empty, don't include it in required
		if hasDef && isEmptyValue(defVal) {
			isDebug := cfg != nil && cfg.Debug
			if isDebug {
				fmt.Printf("Post-processing: Removing field from required list because it has an empty default value\n")
			}
			continue
		}

		// Check if this is a component that can be enabled/disabled
		propObj, isObj := defaults[fieldName].(map[string]any)
		if isObj {
			// Check if this component has an 'enabled' field
			if enabled, hasEnabled := propObj["enabled"]; hasEnabled {
				if enabledBool, isBool := enabled.(bool); isBool && !enabledBool {
					isDebug := cfg != nil && cfg.Debug
					if isDebug {
						fmt.Printf("Post-processing: Removing field from required list because it is disabled\n")
					}
					continue
				}
			}

			// Also check if the component has a nil value by default
			if isEmptyValue(propObj) {
				isDebug := cfg != nil && cfg.Debug
				if isDebug {
					fmt.Printf("Post-processing: Removing field from required list because it has a nil default value\n")
				}
				continue
			}
		}

		// If we reach here, keep the field in the required list
		newRequired = append(newRequired, fieldName)
	}

	// Replace the required list with our filtered one
	if len(newRequired) > 0 {
		schema["required"] = newRequired
	} else {
		delete(schema, "required")
	}

	// Process each property recursively
	for name, prop := range properties {
		propObj, isObj := prop.(map[string]any)
		if !isObj {
			continue
		}

		defVal, hasDef := defaults[name]
		if hasDef {
			defMap, isMap := defVal.(map[string]any)
			if isMap {
				processProperties(propObj, defMap)
			}
		}
	}
}

func NewGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate <context-dir>",
		Short: "Generate JSON Schema from values.yaml",
		Long:  `Generate JSON Schema from values.yaml, optionally merging an overrides YAML file.`,
		Args:  cobra.ExactArgs(1),
		// Do not print usage on error; just show the error message
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := args[0]
			// Validate overrides file if provided
			overridesFlag, err := cmd.Flags().GetString("overrides")
			if err != nil {
				return err
			}
			if overridesFlag != "" {
				overridePath := filepath.Join(ctx, overridesFlag)
				if _, err := os.Stat(overridePath); err != nil {
					return fmt.Errorf("overrides file %s not found in %s", overridesFlag, ctx)
				}
			}
			msg, err := Generate(ctx, overridesFlag)
			if err != nil {
				return err
			}
			fmt.Println(msg)
			return nil
		},
	}
	cmd.Flags().StringP("overrides", "f", "", "path (relative to context dir) to overrides YAML (optional)")
	return cmd
}