package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/mkm29/valet/internal/config"
	"github.com/mkm29/valet/internal/telemetry"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
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
							zap.L().Debug("Skipping field because it has an empty default value",
								zap.String("field", k),
								zap.String("type", fmt.Sprintf("%T", vDefault)))
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
func Generate(ctxDir, overridesFlag string) (string, error) {
	// Create context for tracing
	ctx := context.Background()
	tel := GetTelemetry()

	// Start main span
	start := time.Now()
	ctx, span := tel.StartSpan(ctx, "generate.command",
		trace.WithAttributes(
			attribute.String("context_dir", ctxDir),
			attribute.Bool("has_overrides", overridesFlag != ""),
		),
	)
	defer span.End()

	// Function to execute the actual generation
	executeGenerate := func() (string, error) {
		return generateInternal(ctx, tel, ctxDir, overridesFlag)
	}

	// Execute with telemetry wrapper if enabled
	if tel.IsEnabled() {
		result, err := executeGenerate()
		duration := time.Since(start)

		// Record command metrics
		if cmdMetrics, metricsErr := tel.NewCommandMetrics(); metricsErr == nil {
			cmdMetrics.RecordCommandExecution(ctx, "generate", duration, err)
		}

		// Set span status
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "Schema generated successfully")
		}

		return result, err
	}

	// Execute without telemetry
	return executeGenerate()
}

// generateInternal contains the actual generation logic
func generateInternal(ctx context.Context, tel *telemetry.Telemetry, ctxDir, overridesFlag string) (string, error) {
	// Locate values file (values.yaml or values.yml)
	valuesPath := filepath.Join(ctxDir, "values.yaml")
	if _, err := os.Stat(valuesPath); os.IsNotExist(err) {
		alt := filepath.Join(ctxDir, "values.yml")
		if _, err2 := os.Stat(alt); os.IsNotExist(err2) {
			return "", fmt.Errorf("no values.yaml or values.yml found in %s", ctxDir)
		}
		valuesPath = alt
	}
	var overridesPath string
	if overridesFlag != "" {
		overridesPath = filepath.Join(ctxDir, overridesFlag)
	}
	// Load main values file with tracing
	ctx, loadSpan := tel.StartSpan(ctx, "load.values_yaml",
		trace.WithAttributes(attribute.String("file", valuesPath)),
	)
	yaml1, err := loadYAML(valuesPath)
	loadSpan.End()
	if err != nil {
		telemetry.RecordError(ctx, err)
		return "", fmt.Errorf("error loading %s: %w", valuesPath, err)
	}

	// Record file metrics
	if fileMetrics, metricsErr := tel.NewFileOperationMetrics(); metricsErr == nil {
		if fi, statErr := os.Stat(valuesPath); statErr == nil {
			fileMetrics.RecordFileRead(ctx, valuesPath, fi.Size(), nil)
		}
	}

	// Log some of the top-level default values to help with debugging
	// Use safe debugging to handle cases when cfg is nil (testing environment)
	isDebug := cfg != nil && cfg.Debug
	if isDebug && tel.IsEnabled() {
		logger := tel.Logger()
		logger.Debug(ctx, "Original YAML values loaded",
			zap.String("file", valuesPath),
			zap.Int("top_level_keys", len(yaml1)),
		)

		// Count components with enabled field
		enabledComponentCount := 0
		disabledComponentCount := 0
		for key, compVal := range yaml1 {
			if compMap, isMap := compVal.(map[string]any); isMap {
				// Log components with enabled field
				if enabled, hasEnabled := compMap["enabled"]; hasEnabled {
					if enabledBool, isBool := enabled.(bool); isBool {
						if enabledBool {
							enabledComponentCount++
						} else {
							disabledComponentCount++
						}
						logger.Debug(ctx, "Component status",
							zap.String("component", key),
							zap.Bool("enabled", enabledBool),
						)
					}
				}
			}
		}
		logger.Debug(ctx, "Component summary",
			zap.Int("enabled_count", enabledComponentCount),
			zap.Int("disabled_count", disabledComponentCount),
		)
	}

	var merged map[string]any
	if overridesPath != "" {
		// Load overrides file with tracing
		ctx, overrideSpan := tel.StartSpan(ctx, "load.overrides_yaml",
			trace.WithAttributes(attribute.String("file", overridesPath)),
		)
		yaml2, err := loadYAML(overridesPath)
		overrideSpan.End()
		if err != nil {
			telemetry.RecordError(ctx, err)
			return "", fmt.Errorf("error loading %s: %w", overridesPath, err)
		}

		// Merge with tracing
		ctx, mergeSpan := tel.StartSpan(ctx, "merge.yaml_files")
		merged = deepMerge(yaml1, yaml2)
		mergeSpan.End()
	} else {
		merged = yaml1
	}

	// Generate schema with tracing
	ctx, schemaSpan := tel.StartSpan(ctx, "generate.schema")
	schemaStart := time.Now()
	schema := inferSchema(merged, yaml1)
	schema["$schema"] = "http://json-schema.org/schema#"

	// Post-process the schema to ensure no empty fields are in the required lists
	cleanupRequiredFields(schema, yaml1)
	schemaSpan.End()

	// Record schema generation metrics
	if schemaMetrics, metricsErr := tel.NewSchemaGenerationMetrics(); metricsErr == nil {
		fieldCount := countSchemaFields(schema)
		schemaMetrics.RecordSchemaGeneration(ctx, int64(fieldCount), time.Since(schemaStart), nil)
	}

	outPath := filepath.Join(ctxDir, "values.schema.json")

	// Marshal JSON with tracing
	ctx, marshalSpan := tel.StartSpan(ctx, "marshal.json")
	data, err := json.MarshalIndent(schema, "", "  ")
	marshalSpan.End()
	if err != nil {
		telemetry.RecordError(ctx, err)
		return "", fmt.Errorf("error marshaling JSON: %w", err)
	}

	// Write file with tracing
	ctx, writeSpan := tel.StartSpan(ctx, "write.schema_file",
		trace.WithAttributes(
			attribute.String("file", outPath),
			attribute.Int("size", len(data)),
		),
	)
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		writeSpan.End()
		telemetry.RecordError(ctx, err)
		return "", fmt.Errorf("error writing %s: %w", outPath, err)
	}
	writeSpan.End()

	// Record file write metrics
	if fileMetrics, metricsErr := tel.NewFileOperationMetrics(); metricsErr == nil {
		fileMetrics.RecordFileWrite(ctx, outPath, int64(len(data)), nil)
	}

	if overridesPath != "" {
		return fmt.Sprintf("Generated %s by merging %s into values.yaml", outPath, overridesFlag), nil
	}
	return fmt.Sprintf("Generated %s from values.yaml", outPath), nil
}

// countSchemaFields counts the number of fields in a schema recursively
func countSchemaFields(schema map[string]any) int {
	count := 0
	if props, ok := schema["properties"].(map[string]any); ok {
		count += len(props)
		for _, prop := range props {
			if propMap, ok := prop.(map[string]any); ok {
				count += countSchemaFields(propMap)
			}
		}
	}
	return count
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
				zap.L().Debug("Post-processing: Removing required fields from component because it has enabled=false")
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
				zap.L().Debug("Post-processing: Removing field from required list because it has an empty default value",
					zap.String("field", fieldName))
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
						zap.L().Debug("Post-processing: Removing field from required list because it is disabled",
							zap.String("field", fieldName))
					}
					continue
				}
			}

			// Also check if the component has a nil value by default
			if isEmptyValue(propObj) {
				isDebug := cfg != nil && cfg.Debug
				if isDebug {
					zap.L().Debug("Post-processing: Removing field from required list because it has a nil default value",
						zap.String("field", fieldName))
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
		Use:   "generate [context-dir]",
		Short: "Generate JSON Schema from values.yaml",
		Long: `Generate JSON Schema from values.yaml, optionally merging an overrides YAML file.

You can generate a schema from either:
- A local Helm chart directory (provide context-dir)
- A remote Helm chart (use --chart-name and related flags, or helm config in config file)`,
		Args: cobra.MaximumNArgs(1),
		// Do not print usage on error; just show the error message
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get context directory if provided
			var ctx string
			if len(args) > 0 {
				ctx = args[0]
			}

			// Check if this is a remote chart request via CLI flags
			chartName, _ := cmd.Flags().GetString("chart-name")

			// Determine if we have remote chart config (either from flags or config file)
			hasRemoteChartFlags := chartName != ""
			hasRemoteChartConfig := cfg != nil && cfg.Helm != nil && cfg.Helm.Chart != nil && cfg.Helm.Chart.Name != ""
			hasLocalContext := ctx != ""

			// Validate: must have either local context or remote chart config, but not both
			if !hasLocalContext && !hasRemoteChartFlags && !hasRemoteChartConfig {
				return fmt.Errorf("must provide either a context directory for local chart or remote chart configuration (via --chart-name or helm config in config file)")
			}

			if hasLocalContext && (hasRemoteChartFlags || hasRemoteChartConfig) {
				return fmt.Errorf("cannot specify both local context directory and remote chart configuration")
			}

			// Validate overrides file if provided
			overridesFlag, err := cmd.Flags().GetString("overrides")
			if err != nil {
				return err
			}

			// Handle remote chart case
			if hasRemoteChartFlags || hasRemoteChartConfig {
				// Build helm config from flags if provided
				if hasRemoteChartFlags {
					chartVersion, _ := cmd.Flags().GetString("chart-version")
					registryURL, _ := cmd.Flags().GetString("registry-url")
					registryType, _ := cmd.Flags().GetString("registry-type")

					if chartName == "" {
						return fmt.Errorf("--chart-name is required when using remote chart")
					}
					if chartVersion == "" {
						return fmt.Errorf("--chart-version is required when using remote chart")
					}
					if registryURL == "" {
						return fmt.Errorf("--registry-url is required when using remote chart")
					}

					// Build HelmConfig from flags
					helmConfig := &config.HelmConfig{
						Chart: &config.HelmChartConfig{
							Name:    chartName,
							Version: chartVersion,
							Registry: &config.HelmRegistryConfig{
								URL:      registryURL,
								Type:     registryType,
								Insecure: false,
								Auth:     config.NewHelmAuthConfig(),
								TLS:      config.NewHelmTLSConfig(),
							},
						},
					}

					// Get optional flags
					if insecure, _ := cmd.Flags().GetBool("registry-insecure"); cmd.Flags().Changed("registry-insecure") {
						helmConfig.Chart.Registry.Insecure = insecure
					}

					// Authentication flags
					if username, _ := cmd.Flags().GetString("registry-username"); username != "" {
						helmConfig.Chart.Registry.Auth.Username = username
					}
					if password, _ := cmd.Flags().GetString("registry-password"); password != "" {
						helmConfig.Chart.Registry.Auth.Password = password
					}
					if token, _ := cmd.Flags().GetString("registry-token"); token != "" {
						helmConfig.Chart.Registry.Auth.Token = token
					}

					// TLS flags
					if skipVerify, _ := cmd.Flags().GetBool("registry-tls-skip-verify"); cmd.Flags().Changed("registry-tls-skip-verify") {
						helmConfig.Chart.Registry.TLS.InsecureSkipTLSVerify = skipVerify
					}
					if certFile, _ := cmd.Flags().GetString("registry-cert-file"); certFile != "" {
						helmConfig.Chart.Registry.TLS.CertFile = certFile
					}
					if keyFile, _ := cmd.Flags().GetString("registry-key-file"); keyFile != "" {
						helmConfig.Chart.Registry.TLS.KeyFile = keyFile
					}
					if caFile, _ := cmd.Flags().GetString("registry-ca-file"); caFile != "" {
						helmConfig.Chart.Registry.TLS.CaFile = caFile
					}

					// Validate the helm config
					if err := helmConfig.Validate(); err != nil {
						return fmt.Errorf("invalid helm configuration: %w", err)
					}

					// TODO: Use helmConfig to generate schema from remote chart
					return fmt.Errorf("remote chart support not yet implemented")
				} else {
					// Use config file helm configuration
					// TODO: Use cfg.Helm to generate schema from remote chart
					return fmt.Errorf("remote chart support via config file not yet implemented")
				}
			}

			// Handle local chart case
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

	// Remote chart flags
	cmd.Flags().String("chart-name", "", "name of the remote Helm chart")
	cmd.Flags().String("chart-version", "", "version of the remote Helm chart")
	cmd.Flags().String("registry-url", "", "URL of the Helm chart registry")
	cmd.Flags().String("registry-type", "HTTPS", "type of registry (HTTP, HTTPS, OCI)")
	cmd.Flags().Bool("registry-insecure", false, "allow insecure connections to the registry")

	// Authentication flags
	cmd.Flags().String("registry-username", "", "username for registry authentication")
	cmd.Flags().String("registry-password", "", "password for registry authentication")
	cmd.Flags().String("registry-token", "", "token for registry authentication")

	// TLS flags
	cmd.Flags().Bool("registry-tls-skip-verify", false, "skip TLS certificate verification")
	cmd.Flags().String("registry-cert-file", "", "path to client certificate file")
	cmd.Flags().String("registry-key-file", "", "path to client key file")
	cmd.Flags().String("registry-ca-file", "", "path to CA certificate file")

	return cmd
}
