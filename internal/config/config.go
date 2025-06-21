package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// Config holds the configuration for the application
type Config struct {
	Debug     bool             `yaml:"debug"`
	Context   string           `yaml:"context"`
	Overrides string           `yaml:"overrides"`
	Output    string           `yaml:"output"`
	Telemetry *TelemetryConfig `yaml:"telemetry"`
	Helm      *HelmConfig      `yaml:"helm"`
}

// HelmConfig holds the configuration for Helm chart operations
type HelmConfig struct {
	// Chart specifies the remote chart details
	Chart *HelmChartConfig `yaml:"chart"`
}

// HelmChartConfig holds the configuration for a specific Helm chart
type HelmChartConfig struct {
	// Name is the chart name
	Name string `yaml:"name"`
	// Version is the chart version
	Version string `yaml:"version"`
	// Registry holds the registry configuration
	Registry *HelmRegistryConfig `yaml:"registry"`
}

// HelmRegistryConfig holds the registry configuration
type HelmRegistryConfig struct {
	// URL is the registry URL
	URL string `yaml:"url"`
	// Type is the registry type (HTTP, HTTPS, OCI)
	Type string `yaml:"type"`
	// Insecure allows insecure connections
	Insecure bool `yaml:"insecure"`
	// Auth holds authentication configuration
	Auth *HelmAuthConfig `yaml:"auth"`
	// TLS holds TLS configuration
	TLS *HelmTLSConfig `yaml:"tls"`
}

// HelmAuthConfig holds authentication configuration
type HelmAuthConfig struct {
	// Username for basic auth
	Username string `yaml:"username"`
	// Password for basic auth
	Password string `yaml:"password"`
	// Token for token-based auth
	Token string `yaml:"token"`
}

// HelmTLSConfig holds TLS configuration
type HelmTLSConfig struct {
	// InsecureSkipTLSVerify skips TLS verification
	InsecureSkipTLSVerify bool `yaml:"insecureSkipTLSVerify"`
	// CertFile is the path to the client certificate
	CertFile string `yaml:"certFile"`
	// KeyFile is the path to the client key
	KeyFile string `yaml:"keyFile"`
	// CaFile is the path to the CA certificate
	CaFile string `yaml:"caFile"`
}

// TelemetryConfig holds the telemetry configuration
type TelemetryConfig struct {
	// Enabled determines if telemetry is enabled
	Enabled bool `yaml:"enabled"`
	// ServiceName overrides the default service name
	ServiceName string `yaml:"serviceName"`
	// ServiceVersion overrides the default service version
	ServiceVersion string `yaml:"serviceVersion"`
	// ExporterType determines the exporter type (otlp, stdout, none)
	ExporterType string `yaml:"exporterType"`
	// OTLPEndpoint is the OTLP endpoint for traces and metrics
	OTLPEndpoint string `yaml:"otlpEndpoint"`
	// Insecure determines if the OTLP connection should be insecure
	Insecure bool `yaml:"insecure"`
	// Headers are additional headers to send with OTLP requests
	Headers map[string]string `yaml:"headers"`
	// SampleRate is the trace sampling rate (0.0 to 1.0)
	SampleRate float64 `yaml:"sampleRate"`
}

// NewTelemetryConfig returns the default telemetry configuration
func NewTelemetryConfig() *TelemetryConfig {
	return &TelemetryConfig{
		Enabled:        false,
		ServiceName:    "valet",
		ServiceVersion: "0.1.0",
		ExporterType:   "none",
		OTLPEndpoint:   "localhost:4317",
		Insecure:       true,
		Headers:        make(map[string]string),
		SampleRate:     1.0,
	}
}

// NewHelmConfig returns the default Helm configuration
func NewHelmConfig() *HelmConfig {
	return &HelmConfig{
		Chart: nil,
	}
}

// NewHelmChartConfig returns the default Helm chart configuration
func NewHelmChartConfig() *HelmChartConfig {
	return &HelmChartConfig{
		Name:     "",
		Version:  "",
		Registry: NewHelmRegistryConfig(),
	}
}

// NewHelmRegistryConfig returns the default Helm registry configuration
func NewHelmRegistryConfig() *HelmRegistryConfig {
	return &HelmRegistryConfig{
		URL:      "",
		Type:     "HTTPS",
		Insecure: false,
		Auth:     NewHelmAuthConfig(),
		TLS:      NewHelmTLSConfig(),
	}
}

// NewHelmAuthConfig returns the default Helm auth configuration
func NewHelmAuthConfig() *HelmAuthConfig {
	return &HelmAuthConfig{
		Username: "",
		Password: "",
		Token:    "",
	}
}

// NewHelmTLSConfig returns the default Helm TLS configuration
func NewHelmTLSConfig() *HelmTLSConfig {
	return &HelmTLSConfig{
		InsecureSkipTLSVerify: false,
		CertFile:              "",
		KeyFile:               "",
		CaFile:                "",
	}
}

// validate TelemetryConfig struct
func (c *TelemetryConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("telemetry config is nil")
	}
	if c.ExporterType != "otlp" && c.ExporterType != "stdout" && c.ExporterType != "none" {
		return fmt.Errorf("invalid exporter type: %s", c.ExporterType)
	}
	if c.SampleRate < 0.0 || c.SampleRate > 1.0 {
		return fmt.Errorf("sample rate must be between 0.0 and 1.0, got: %f", c.SampleRate)
	}
	return nil
}

// Validate validates the HelmConfig
func (h *HelmConfig) Validate() error {
	if h == nil || h.Chart == nil {
		return nil // Helm config is optional
	}

	if h.Chart.Name == "" {
		return fmt.Errorf("helm chart name is required")
	}
	if h.Chart.Version == "" {
		return fmt.Errorf("helm chart version is required")
	}

	if h.Chart.Registry == nil {
		return fmt.Errorf("helm registry configuration is required")
	}

	if h.Chart.Registry.URL == "" {
		return fmt.Errorf("helm registry URL is required")
	}

	// Validate registry type
	validTypes := map[string]bool{"HTTP": true, "HTTPS": true, "OCI": true}
	if !validTypes[h.Chart.Registry.Type] {
		return fmt.Errorf("invalid registry type: %s (must be HTTP, HTTPS, or OCI)", h.Chart.Registry.Type)
	}

	// Validate TLS configuration
	if h.Chart.Registry.TLS != nil {
		// If cert file is provided, key file must also be provided
		if (h.Chart.Registry.TLS.CertFile != "" && h.Chart.Registry.TLS.KeyFile == "") ||
			(h.Chart.Registry.TLS.CertFile == "" && h.Chart.Registry.TLS.KeyFile != "") {
			return fmt.Errorf("both cert file and key file must be provided for client TLS")
		}
	}

	return nil
}

// LoadConfig reads configuration from a YAML file (if it exists).
// If the file is not found, returns an empty Config without error.
func LoadConfig(path string) (*Config, error) {
	cfg := &Config{
		Telemetry: NewTelemetryConfig(),
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Ensure telemetry config is not nil
	if cfg.Telemetry == nil {
		cfg.Telemetry = NewTelemetryConfig()
	}

	// Apply defaults to Helm configuration if present
	if cfg.Helm != nil && cfg.Helm.Chart != nil {
		if cfg.Helm.Chart.Registry != nil {
			// Apply default registry type if not specified
			if cfg.Helm.Chart.Registry.Type == "" {
				cfg.Helm.Chart.Registry.Type = "HTTPS"
			}
			// Ensure Auth and TLS are not nil
			if cfg.Helm.Chart.Registry.Auth == nil {
				cfg.Helm.Chart.Registry.Auth = NewHelmAuthConfig()
			}
			if cfg.Helm.Chart.Registry.TLS == nil {
				cfg.Helm.Chart.Registry.TLS = NewHelmTLSConfig()
			}
		}
	}

	return cfg, nil
}

// ToHelmChart converts the HelmConfig to a helm.Chart structure
func (h *HelmConfig) ToHelmChart() *HelmChart {
	if h == nil || h.Chart == nil {
		return nil
	}

	chart := &HelmChart{
		Name:    h.Chart.Name,
		Version: h.Chart.Version,
	}

	if h.Chart.Registry != nil {
		registry := &HelmRegistry{
			URL:      h.Chart.Registry.URL,
			Type:     h.Chart.Registry.Type,
			Insecure: h.Chart.Registry.Insecure,
		}

		if h.Chart.Registry.Auth != nil {
			registry.Auth.Username = h.Chart.Registry.Auth.Username
			registry.Auth.Password = h.Chart.Registry.Auth.Password
			registry.Auth.Token = h.Chart.Registry.Auth.Token
		}

		if h.Chart.Registry.TLS != nil {
			registry.TLS.InsecureSkipTLSVerify = h.Chart.Registry.TLS.InsecureSkipTLSVerify
			registry.TLS.CertFile = h.Chart.Registry.TLS.CertFile
			registry.TLS.KeyFile = h.Chart.Registry.TLS.KeyFile
			registry.TLS.CaFile = h.Chart.Registry.TLS.CaFile
		}

		chart.Registry = registry
	}

	return chart
}

// HelmChart represents a Helm chart (for internal use)
type HelmChart struct {
	Registry *HelmRegistry
	Name     string
	Version  string
}

// HelmRegistry represents a Helm registry (for internal use)
type HelmRegistry struct {
	URL  string
	Auth struct {
		Username string
		Password string
		Token    string
	}
	TLS struct {
		InsecureSkipTLSVerify bool
		CertFile              string
		KeyFile               string
		CaFile                string
	}
	Insecure bool
	Type     string
}
