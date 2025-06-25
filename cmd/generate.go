package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/mkm29/valet/internal/helm"
	"github.com/mkm29/valet/internal/telemetry"
	"github.com/mkm29/valet/internal/utils"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// GenerateWithApp is a version of Generate that accepts dependencies
func GenerateWithApp(app *App, ctxDir, overridesFlag string) (string, error) {
	// Create context for tracing
	ctx := context.Background()
	tel := app.Telemetry
	if tel == nil {
		// Create a no-op telemetry if not provided
		tel = &telemetry.Telemetry{}
	}

	// Generate a request ID for correlation
	requestID := utils.GenerateRequestID()
	ctx = telemetry.EnrichContextWithRequestID(ctx, requestID)

	// Set sampling priority for command execution
	ctx = telemetry.EnrichContextWithSamplingPriority(ctx, telemetry.SamplingPriorityAccept)

	// Start main span
	start := time.Now()
	ctx, span := tel.StartSpan(ctx, "generate.command",
		trace.WithAttributes(
			attribute.String("context_dir", ctxDir),
			attribute.Bool("has_overrides", overridesFlag != ""),
			attribute.String("request.id", requestID),
		),
	)
	defer span.End()

	// Function to execute the actual generation
	executeGenerate := func() (string, error) {
		return generateInternalWithApp(ctx, app, tel, ctxDir, overridesFlag)
	}

	// Execute with telemetry wrapper if enabled
	if tel.IsEnabled() {
		result, err := executeGenerate()
		duration := time.Since(start)

		// Record command metrics to both OpenTelemetry and Prometheus
		tel.RecordCommandExecutionWithServer(ctx, "generate", duration, err)

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

// generateInternalWithApp contains the actual generation logic with dependency injection
func generateInternalWithApp(ctx context.Context, app *App, tel *telemetry.Telemetry, ctxDir, overridesFlag string) (string, error) {
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
	yaml1, err := utils.LoadYAML(valuesPath)
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
	isDebug := app.Config != nil && app.Config.LogLevel.Level == slog.LevelDebug
	if isDebug && tel.IsEnabled() && app.Logger != nil {
		logDefaultValues(ctx, app.Logger, valuesPath, yaml1)
	}

	var merged map[string]any
	if overridesPath != "" {
		// Load overrides file with tracing
		ctx, overrideSpan := tel.StartSpan(ctx, "load.overrides_yaml",
			trace.WithAttributes(attribute.String("file", overridesPath)),
		)
		yaml2, err := utils.LoadYAML(overridesPath)
		overrideSpan.End()
		if err != nil {
			telemetry.RecordError(ctx, err)
			return "", fmt.Errorf("error loading %s: %w", overridesPath, err)
		}

		// Merge with tracing
		_, mergeSpan := tel.StartSpan(ctx, "merge.yaml_files")
		merged = utils.DeepMerge(yaml1, yaml2)
		mergeSpan.End()
	} else {
		merged = yaml1
	}

	// Generate schema with tracing
	ctx, schemaSpan := tel.StartSpan(ctx, "generate.schema")
	schemaStart := time.Now()
	schema := inferSchema(app, merged, yaml1)
	schema["$schema"] = "http://json-schema.org/schema#"

	// Post-process the schema to ensure no empty fields are in the required lists
	cleanupRequiredFieldsWithApp(app, schema, yaml1)
	schemaSpan.End()

	// Record schema generation metrics
	if schemaMetrics, metricsErr := tel.NewSchemaGenerationMetrics(); metricsErr == nil {
		fieldCount := utils.CountSchemaFields(schema)
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

// logDefaultValues logs debugging information about default values
func logDefaultValues(_ context.Context, logger *slog.Logger, valuesPath string, yaml1 map[string]any) {
	logger.Debug("Original YAML values loaded",
		"file", valuesPath,
		"top_level_keys", len(yaml1),
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
					logger.Debug("Component status",
						"component", key,
						"enabled", enabledBool,
					)
				}
			}
		}
	}
	logger.Debug("Component summary",
		"enabled_count", enabledComponentCount,
		"disabled_count", disabledComponentCount,
	)
}

// inferSchema builds a JSON‐Schema fragment with dependency injection
func inferSchema(app *App, val, defaultVal any) map[string]any {
	switch v := val.(type) {
	case map[string]any:
		return inferObjectSchemaWithApp(app, v, defaultVal)
	case []any:
		return utils.InferArraySchema(v, defaultVal, func(item, defItem any) map[string]any {
			return inferSchema(app, item, defItem)
		})
	case bool:
		return utils.InferBooleanSchema(v)
	case int, int64:
		return utils.InferIntegerSchema(v)
	case float64:
		return utils.InferNumberSchema(v)
	case string:
		return utils.InferStringSchema(v)
	default:
		return inferUnknownTypeSchemaWithApp(app, v, defaultVal)
	}
}

// inferObjectSchemaWithApp processes map[string]any types with dependency injection
func inferObjectSchemaWithApp(app *App, v map[string]any, defaultVal any) map[string]any {
	defMap, _ := defaultVal.(map[string]any)

	// Generate properties
	props := generateObjectPropertiesWithApp(app, v, defMap)

	// Build defaults
	defaults := utils.BuildObjectDefaults(v)

	// Create base schema
	schema := map[string]any{
		"type":       "object",
		"properties": props,
		"default":    defaults,
	}

	// Determine required fields
	required := determineRequiredFieldsWithApp(app, v, defMap)
	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

// generateObjectPropertiesWithApp recursively generates schema properties
func generateObjectPropertiesWithApp(app *App, v map[string]any, defMap map[string]any) map[string]any {
	props := make(map[string]any, len(v))
	for key, sub := range v {
		// Ensure we pass the correct default value for the subfield
		var defSubVal any
		if defMap != nil {
			defSubVal = defMap[key]
		}
		props[key] = inferSchema(app, sub, defSubVal)
	}
	return props
}

// determineRequiredFieldsWithApp determines required fields with dependency injection
func determineRequiredFieldsWithApp(app *App, v, defMap map[string]any) []string {
	var required []string
	for k, vDefault := range defMap {
		// Only add to required if key exists, default is NOT nil and NOT null
		if _, exists := v[k]; exists {
			if shouldFieldBeRequiredWithApp(app, k, v[k], vDefault, v) {
				required = append(required, k)
			}
		}
	}
	return required
}

// shouldFieldBeRequiredWithApp checks if a field should be required with dependency injection
func shouldFieldBeRequiredWithApp(app *App, fieldName string, fieldValue, defaultValue any, parentObject map[string]any) bool {
	// Check if default is nil or null string
	if utils.IsNullValue(defaultValue) {
		return false
	}

	// Check for empty values
	if utils.IsEmptyValue(defaultValue) {
		isDebug := app.Config != nil && app.Config.LogLevel.Level == slog.LevelDebug
		if isDebug && app.Logger != nil {
			app.Logger.Debug("Skipping field because it has an empty default value",
				"field", fieldName,
				"type", fmt.Sprintf("%T", defaultValue))
		}
		return false
	}

	// Check if this is a component that can be enabled/disabled
	if utils.IsDisabledComponent(fieldValue) {
		return false
	}

	// Check if parent component is disabled
	if utils.IsChildOfDisabledComponent(parentObject) {
		return false
	}

	return true
}

// inferUnknownTypeSchemaWithApp handles unknown types with dependency injection
func inferUnknownTypeSchemaWithApp(app *App, v any, defaultVal any) map[string]any {
	rv := reflect.ValueOf(v)

	// Handle nil values
	if !rv.IsValid() || (rv.Kind() == reflect.Ptr && rv.IsNil()) {
		return utils.InferNullSchema()
	}

	// Get the underlying value if it's a pointer
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Map:
		return inferReflectedMapSchemaWithApp(app, rv, defaultVal)
	case reflect.Slice, reflect.Array:
		return inferReflectedArraySchemaWithApp(app, rv, defaultVal)
	case reflect.Bool:
		return utils.InferBooleanSchema(rv.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return utils.InferIntegerSchema(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return utils.InferIntegerSchema(rv.Uint())
	case reflect.Float32, reflect.Float64:
		return utils.InferReflectedFloatSchema(rv.Float())
	case reflect.String:
		return utils.InferStringSchema(rv.String())
	default:
		// Fall back to string representation for other types
		return map[string]any{
			"type":    "string",
			"default": fmt.Sprintf("%v", v),
		}
	}
}

// inferReflectedMapSchemaWithApp handles reflected map types with dependency injection
func inferReflectedMapSchemaWithApp(app *App, rv reflect.Value, defaultVal any) map[string]any {
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
		props[k] = inferSchema(app, sub, defVal)
	}

	return map[string]any{
		"type":       "object",
		"properties": props,
		"default":    defMap,
	}
}

// inferReflectedArraySchemaWithApp handles reflected array/slice types with dependency injection
func inferReflectedArraySchemaWithApp(app *App, rv reflect.Value, defaultVal any) map[string]any {
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
		itemsSchema = inferSchema(app, items[0], defItem)
	} else {
		itemsSchema = map[string]any{}
	}

	return map[string]any{
		"type":    "array",
		"items":   itemsSchema,
		"default": items,
	}
}

// cleanupRequiredFieldsWithApp processes the schema with dependency injection
func cleanupRequiredFieldsWithApp(app *App, schema map[string]any, defaults map[string]any) {
	// Process the top-level schema
	processPropertiesWithApp(app, schema, defaults)
}

// processPropertiesWithApp handles each property with dependency injection
func processPropertiesWithApp(app *App, schema map[string]any, defaults map[string]any) {
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
			isDebug := app.Config != nil && app.Config.LogLevel.Level == slog.LevelDebug
			if isDebug && app.Logger != nil {
				app.Logger.Debug("Post-processing: Removing required fields from component because it has enabled=false")
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
		if hasDef && utils.IsEmptyValue(defVal) {
			isDebug := app.Config != nil && app.Config.LogLevel.Level == slog.LevelDebug
			if isDebug && app.Logger != nil {
				app.Logger.Debug("Post-processing: Removing field from required list because it has an empty default value",
					"field", fieldName)
			}
			continue
		}

		// Check if this is a component that can be enabled/disabled
		propObj, isObj := defaults[fieldName].(map[string]any)
		if isObj {
			// Check if this component has an 'enabled' field
			if enabled, hasEnabled := propObj["enabled"]; hasEnabled {
				if enabledBool, isBool := enabled.(bool); isBool && !enabledBool {
					isDebug := app.Config != nil && app.Config.LogLevel.Level == slog.LevelDebug
					if isDebug && app.Logger != nil {
						app.Logger.Debug("Post-processing: Removing field from required list because it is disabled",
							"field", fieldName)
					}
					continue
				}
			}

			// Also check if the component has a nil value by default
			if utils.IsEmptyValue(propObj) {
				isDebug := app.Config != nil && app.Config.LogLevel.Level == slog.LevelDebug
				if isDebug && app.Logger != nil {
					app.Logger.Debug("Post-processing: Removing field from required list because it has a nil default value",
						"field", fieldName)
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
				processPropertiesWithApp(app, propObj, defMap)
			}
		}
	}
}

// NewGenerateCmdWithApp creates generate command with dependency injection
func NewGenerateCmdWithApp(app *App) *cobra.Command {
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
			// Get app from context if not provided
			if app == nil {
				app = GetAppFromContext(cmd)
			}
			return generateCmdRunWithApp(cmd, args, app)
		},
	}
	addGenerateFlags(cmd)
	return cmd
}

// generateCmdRunWithApp is the main execution function with dependency injection
func generateCmdRunWithApp(cmd *cobra.Command, args []string, app *App) error {
	// Get context directory if provided
	ctx, err := utils.GetContextDirectory(args)
	if err != nil {
		return fmt.Errorf("failed to determine context directory: %w", err)
	}

	// Parse command configuration
	// Pass args to determine if context was explicitly provided
	cmdConfig, err := parseGenerateCommandConfigWithApp(cmd, ctx, app, args)
	if err != nil {
		return err
	}

	// Debug output
	logGenerateCommandDebugWithApp(cmdConfig, app)

	// Validate command configuration
	if err := validateGenerateCommandConfig(cmdConfig); err != nil {
		return err
	}

	// Handle remote or local chart generation
	if cmdConfig.isRemote {
		return handleRemoteChartGenerationWithApp(cmd, cmdConfig, app)
	}

	return handleLocalChartGenerationWithApp(cmdConfig, app)
}

// parseGenerateCommandConfigWithApp parses configuration with dependency injection
func parseGenerateCommandConfigWithApp(cmd *cobra.Command, ctx string, app *App, args []string) (*generateCommandConfig, error) {
	chartName, _ := cmd.Flags().GetString("chart-name")
	overridesFlag, err := cmd.Flags().GetString("overrides")
	if err != nil {
		return nil, err
	}

	config := &generateCommandConfig{
		ctx:                  ctx,
		chartName:            chartName,
		overridesFlag:        overridesFlag,
		hasRemoteChartFlags:  chartName != "",
		hasRemoteChartConfig: app.Config != nil && app.Config.Helm != nil && app.Config.Helm.Chart != nil && app.Config.Helm.Chart.Name != "",
		hasLocalContext:      len(args) > 0, // Only true if context directory was explicitly provided
	}

	config.isRemote = config.hasRemoteChartFlags || config.hasRemoteChartConfig

	return config, nil
}

// logGenerateCommandDebugWithApp logs debug information with dependency injection
func logGenerateCommandDebugWithApp(cmdConfig *generateCommandConfig, app *App) {
	if app.Config != nil && app.Config.LogLevel.Level == slog.LevelDebug && app.Logger != nil {
		app.Logger.Debug("Generate command configuration",
			"hasRemoteChartFlags", cmdConfig.hasRemoteChartFlags,
			"hasRemoteChartConfig", cmdConfig.hasRemoteChartConfig,
			"hasLocalContext", cmdConfig.hasLocalContext,
			"chartName", cmdConfig.chartName,
			"helmConfig", app.Config.Helm,
		)
	}
}

// handleRemoteChartGenerationWithApp handles remote chart generation with dependency injection
func handleRemoteChartGenerationWithApp(cmd *cobra.Command, cmdConfig *generateCommandConfig, app *App) error {
	if cmdConfig.hasRemoteChartFlags {
		return handleRemoteChartFromFlags(cmd)
	}
	return handleRemoteChartFromConfigWithApp(app)
}

// handleRemoteChartFromConfigWithApp uses config file helm configuration with dependency injection
func handleRemoteChartFromConfigWithApp(app *App) error {
	// Create Helm instance with options
	h := helm.NewHelm(helm.HelmOptions{
		Debug: app.Config.LogLevel.Level == slog.LevelDebug,
		// Logger will use the default slog logger
	})

	// First check if the chart has a schema
	hasSchema, err := h.HasSchema(app.Config.Helm.Chart)
	if err != nil {
		errMsg := fmt.Sprintf("error checking for schema in remote chart %s/%s: %v",
			app.Config.Helm.Chart.Name, app.Config.Helm.Chart.Version, err)
		errMsg += "\n\nPlease check your helm configuration in the config file"
		return fmt.Errorf("%s", errMsg)
	}

	if !hasSchema {
		errMsg := fmt.Sprintf("no values.schema.json found in chart %s/%s",
			app.Config.Helm.Chart.Name, app.Config.Helm.Chart.Version)
		errMsg += "\n\nThis chart does not include a JSON schema file."
		errMsg += "\nYou may need to:"
		errMsg += "\n- Create a schema manually based on the chart's values.yaml"
		errMsg += "\n- Check if a newer version of the chart includes a schema"
		errMsg += "\n- Contact the chart maintainer to request a schema be added"
		return fmt.Errorf("%s", errMsg)
	}

	// Download the schema
	loc, cleanup, err := h.DownloadSchema(app.Config.Helm.Chart)
	if err != nil {
		errMsg := fmt.Sprintf("error downloading schema from chart %s/%s: %v",
			app.Config.Helm.Chart.Name, app.Config.Helm.Chart.Version, err)
		return fmt.Errorf("%s", errMsg)
	}
	defer cleanup() // Clean up the temporary file

	// Print out the location of the downloaded schema
	fmt.Printf("Downloaded remote chart schema to: %s\n", loc)
	return nil
}

// handleLocalChartGenerationWithApp handles local chart generation with dependency injection
func handleLocalChartGenerationWithApp(cmdConfig *generateCommandConfig, app *App) error {
	// Validate overrides file if provided
	if err := validateOverridesFile(cmdConfig.ctx, cmdConfig.overridesFlag); err != nil {
		return err
	}

	msg, err := GenerateWithApp(app, cmdConfig.ctx, cmdConfig.overridesFlag)
	if err != nil {
		return err
	}
	fmt.Println(msg)
	return nil
}
