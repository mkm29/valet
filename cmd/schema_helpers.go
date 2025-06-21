package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime/debug"

	"github.com/mkm29/valet/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// Schema generation helper functions

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

// buildNestedDefaults builds defaults for nested map values
func buildNestedDefaults(mapVal map[string]any) map[string]any {
	nestedDefaults := make(map[string]any)
	for k, v := range mapVal {
		if v != nil {
			nestedDefaults[k] = v
		}
	}
	return nestedDefaults
}

// buildObjectDefaults builds the default values for an object schema
func buildObjectDefaults(v map[string]any) map[string]any {
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
			nestedDefaults := buildNestedDefaults(mapVal)
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

// isNullValue checks if a value represents null
func isNullValue(val any) bool {
	if val == nil {
		return true
	}

	if strVal, ok := val.(string); ok {
		return strVal == "null" || strVal == ""
	}

	return false
}

// isDisabledComponent checks if a component has enabled=false
func isDisabledComponent(val any) bool {
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

// isChildOfDisabledComponent checks if the parent object has enabled=false
func isChildOfDisabledComponent(parentObject map[string]any) bool {
	if parent, ok := parentObject["enabled"]; ok {
		if parentEnabled, ok := parent.(bool); ok && !parentEnabled {
			return true
		}
	}
	return false
}

// inferArraySchema processes array types and generates array schema
func inferArraySchema(app *App, v []any, defaultVal any) map[string]any {
	var defItem any
	if defArr, ok := defaultVal.([]any); ok && len(defArr) > 0 {
		defItem = defArr[0]
	}

	var itemsSchema map[string]any
	if len(v) > 0 {
		itemsSchema = inferSchema(app, v[0], defItem)
	} else {
		itemsSchema = map[string]any{}
	}

	return map[string]any{
		"type":    "array",
		"items":   itemsSchema,
		"default": v,
	}
}

// inferBooleanSchema processes boolean types
func inferBooleanSchema(v bool) map[string]any {
	return map[string]any{
		"type":    "boolean",
		"default": v,
	}
}

// inferIntegerSchema processes integer types
func inferIntegerSchema(v any) map[string]any {
	return map[string]any{
		"type":    "integer",
		"default": v,
	}
}

// inferNumberSchema processes float64 types, converting to integer if appropriate
func inferNumberSchema(v float64) map[string]any {
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

// inferStringSchema processes string types, handling null strings specially
func inferStringSchema(v string) map[string]any {
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

// inferNullSchema returns a schema for null values
func inferNullSchema() map[string]any {
	typeArray := []string{"string", "null"}
	return map[string]any{
		"type":    typeArray,
		"default": nil,
	}
}

// inferReflectedFloatSchema handles reflected float types
func inferReflectedFloatSchema(floatVal float64) map[string]any {
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

// generateCommandConfig holds parsed command configuration
type generateCommandConfig struct {
	ctx                  string
	chartName            string
	overridesFlag        string
	hasRemoteChartFlags  bool
	hasRemoteChartConfig bool
	hasLocalContext      bool
	isRemote             bool
}

// getContextDirectory extracts the context directory from arguments
func getContextDirectory(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return ""
}

// validateGenerateCommandConfig validates the command configuration
func validateGenerateCommandConfig(cmdConfig *generateCommandConfig) error {
	// Must have either local context or remote chart config, but not both
	if !cmdConfig.hasLocalContext && !cmdConfig.hasRemoteChartFlags && !cmdConfig.hasRemoteChartConfig {
		return fmt.Errorf("must provide either a context directory for local chart or remote chart configuration (via --chart-name or helm config in config file)")
	}

	if cmdConfig.hasLocalContext && (cmdConfig.hasRemoteChartFlags || cmdConfig.hasRemoteChartConfig) {
		return fmt.Errorf("cannot specify both local context directory and remote chart configuration")
	}

	return nil
}

// validateOverridesFile validates that the overrides file exists if specified
func validateOverridesFile(ctx, overridesFlag string) error {
	if overridesFlag != "" {
		overridePath := filepath.Join(ctx, overridesFlag)
		if _, err := os.Stat(overridePath); err != nil {
			return fmt.Errorf("overrides file %s not found in %s", overridesFlag, ctx)
		}
	}
	return nil
}

// handleRemoteChartFromFlags builds helm config from CLI flags and processes remote chart
func handleRemoteChartFromFlags(cmd *cobra.Command) error {
	helmConfig, err := buildHelmConfigFromFlags(cmd)
	if err != nil {
		return err
	}

	// Validate the helm config
	if err := helmConfig.Validate(); err != nil {
		return fmt.Errorf("invalid helm configuration: %w", err)
	}

	// TODO: Use helmConfig to generate schema from remote chart
	return fmt.Errorf("remote chart support not yet implemented")
}

// buildHelmConfigFromFlags creates HelmConfig from command flags
func buildHelmConfigFromFlags(cmd *cobra.Command) (*config.HelmConfig, error) {
	// Get required flags
	chartName, _ := cmd.Flags().GetString("chart-name")
	chartVersion, _ := cmd.Flags().GetString("chart-version")
	registryURL, _ := cmd.Flags().GetString("registry-url")
	registryType, _ := cmd.Flags().GetString("registry-type")

	// Validate required flags
	if err := validateRequiredRemoteFlags(chartName, chartVersion, registryURL); err != nil {
		return nil, err
	}

	// Build base config
	helmConfig := &config.HelmConfig{
		Chart: &config.HelmChart{
			Name:    chartName,
			Version: chartVersion,
			Registry: &config.HelmRegistry{
				URL:      registryURL,
				Type:     registryType,
				Insecure: false,
				Auth:     config.NewHelmAuth(),
				TLS:      config.NewHelmTLS(),
			},
		},
	}

	// Apply optional flags
	applyOptionalHelmFlags(cmd, helmConfig)

	return helmConfig, nil
}

// validateRequiredRemoteFlags validates required flags for remote chart
func validateRequiredRemoteFlags(chartName, chartVersion, registryURL string) error {
	if chartName == "" {
		return fmt.Errorf("--chart-name is required when using remote chart")
	}
	if chartVersion == "" {
		return fmt.Errorf("--chart-version is required when using remote chart")
	}
	if registryURL == "" {
		return fmt.Errorf("--registry-url is required when using remote chart")
	}
	return nil
}

// applyOptionalHelmFlags applies optional flags to helm configuration
func applyOptionalHelmFlags(cmd *cobra.Command, helmConfig *config.HelmConfig) {
	// Registry settings
	if insecure, _ := cmd.Flags().GetBool("registry-insecure"); cmd.Flags().Changed("registry-insecure") {
		helmConfig.Chart.Registry.Insecure = insecure
	}

	// Authentication
	applyAuthenticationFlags(cmd, helmConfig)

	// TLS settings
	applyTLSFlags(cmd, helmConfig)
}

// applyAuthenticationFlags applies authentication-related flags
func applyAuthenticationFlags(cmd *cobra.Command, helmConfig *config.HelmConfig) {
	if username, _ := cmd.Flags().GetString("registry-username"); username != "" {
		helmConfig.Chart.Registry.Auth.Username = username
	}
	if password, _ := cmd.Flags().GetString("registry-password"); password != "" {
		helmConfig.Chart.Registry.Auth.Password = password
	}
	if token, _ := cmd.Flags().GetString("registry-token"); token != "" {
		helmConfig.Chart.Registry.Auth.Token = token
	}
}

// applyTLSFlags applies TLS-related flags
func applyTLSFlags(cmd *cobra.Command, helmConfig *config.HelmConfig) {
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
}

// addGenerateFlags adds all flags to the generate command
func addGenerateFlags(cmd *cobra.Command) {
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
}

// GetBuildVersion returns the build version information
func GetBuildVersion() string {
	// Default version when running directly with `go run`
	version := "(unknown version)"
	commit := "(unknown commit)"

	// Read build info
	info, ok := debug.ReadBuildInfo()
	if ok {
		// Extract version from module
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}

		// Extract commit from VCS settings
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				commit = setting.Value
				break
			}
		}
	}

	// Format output
	if commit != "(unknown commit)" {
		return fmt.Sprintf("%s@%s (commit %s)", info.Main.Path, version, commit)
	}
	return fmt.Sprintf("%s@%s", info.Main.Path, version)
}
