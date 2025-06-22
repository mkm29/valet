package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mkm29/valet/internal/helm"
	"github.com/spf13/cobra"
)

// Schema generation helper functions

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
	helmConfig, err := helm.BuildHelmConfigFromFlags(cmd)
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
