package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/mkm29/valet/internal/config"
	"github.com/spf13/cobra"
)

// Schema generation helper functions

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
	errMsg := "remote chart support via CLI flags is not yet fully implemented"
	errMsg += "\n\nAs a workaround, you can:"
	errMsg += "\n1. Create a config file (.valet.yaml) with your helm configuration"
	errMsg += "\n2. Use the config file approach: valet generate --config-file .valet.yaml"
	errMsg += "\n\nExample config file:"
	errMsg += "\n```yaml"
	errMsg += "\nhelm:"
	errMsg += "\n  chart:"
	errMsg += fmt.Sprintf("\n    name: %s", helmConfig.Chart.Name)
	errMsg += fmt.Sprintf("\n    version: %s", helmConfig.Chart.Version)
	errMsg += "\n    registry:"
	errMsg += fmt.Sprintf("\n      url: %s", helmConfig.Chart.Registry.URL)
	errMsg += fmt.Sprintf("\n      type: %s", helmConfig.Chart.Registry.Type)
	if helmConfig.Chart.Registry.Auth.Username != "" {
		errMsg += "\n      auth:"
		errMsg += "\n        username: <your-username>"
		errMsg += "\n        password: <your-password>"
	}
	errMsg += "\n```"

	return fmt.Errorf("%s", errMsg)
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
