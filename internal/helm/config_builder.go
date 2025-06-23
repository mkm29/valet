package helm

import (
	"fmt"

	"github.com/mkm29/valet/internal/config"
	"github.com/spf13/cobra"
)

// BuildHelmConfigFromFlags creates HelmConfig from command flags
func BuildHelmConfigFromFlags(cmd *cobra.Command) (*config.HelmConfig, error) {
	// Get required flags
	chartName, _ := cmd.Flags().GetString("chart-name")
	chartVersion, _ := cmd.Flags().GetString("chart-version")
	registryURL, _ := cmd.Flags().GetString("registry-url")
	registryType, _ := cmd.Flags().GetString("registry-type")

	// Validate required flags
	if err := ValidateRequiredRemoteFlags(chartName, chartVersion, registryURL); err != nil {
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
	ApplyOptionalHelmFlags(cmd, helmConfig)

	return helmConfig, nil
}

// ValidateRequiredRemoteFlags validates required flags for remote chart
func ValidateRequiredRemoteFlags(chartName, chartVersion, registryURL string) error {
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

// ApplyOptionalHelmFlags applies optional flags to helm configuration
func ApplyOptionalHelmFlags(cmd *cobra.Command, helmConfig *config.HelmConfig) {
	// Registry settings
	if insecure, _ := cmd.Flags().GetBool("registry-insecure"); cmd.Flags().Changed("registry-insecure") {
		helmConfig.Chart.Registry.Insecure = insecure
	}

	// Authentication
	ApplyAuthenticationFlags(cmd, helmConfig)

	// TLS settings
	ApplyTLSFlags(cmd, helmConfig)
}

// ApplyAuthenticationFlags applies authentication-related flags
func ApplyAuthenticationFlags(cmd *cobra.Command, helmConfig *config.HelmConfig) {
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

// ApplyTLSFlags applies TLS-related flags
func ApplyTLSFlags(cmd *cobra.Command, helmConfig *config.HelmConfig) {
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
