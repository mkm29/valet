package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

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

// HelmConfig holds the configuration for Helm chart operations
type HelmConfig struct {
	Chart *HelmChart `yaml:"chart"`
}

// HelmChart holds the configuration for a specific Helm chart
type HelmChart struct {
	Name     string        `yaml:"name"`
	Version  string        `yaml:"version"`
	Registry *HelmRegistry `yaml:"registry"`
}

// HelmRegistry holds the registry configuration
type HelmRegistry struct {
	URL      string    `yaml:"url"`
	Type     string    `yaml:"type"`     // e.g., "HTTP", "HTTPS", "OCI"
	Insecure bool      `yaml:"insecure"` // Whether to allow insecure connections
	Auth     *HelmAuth `yaml:"auth"`
	TLS      *HelmTLS  `yaml:"tls"`
}

// HelmAuth holds authentication configuration
type HelmAuth struct {
	Username string `yaml:"username"` // Optional username for authentication
	Password string `yaml:"password"` // Optional password for authentication
	Token    string `yaml:"token"`    // Optional authentication token for private registries
}

// HelmTLS holds TLS configuration
type HelmTLS struct {
	InsecureSkipTLSVerify bool   `yaml:"insecureSkipTLSVerify"` // Whether to skip TLS verification
	CertFile              string `yaml:"certFile"`              // Path to the client certificate file
	KeyFile               string `yaml:"keyFile"`               // Path to the client key file
	CaFile                string `yaml:"caFile"`                // Path to the CA certificate file
}

// NewHelmConfig returns the default Helm configuration
func NewHelmConfig() *HelmConfig {
	return &HelmConfig{
		Chart: nil,
	}
}

// NewHelmChart returns the default Helm chart configuration
func NewHelmChart() *HelmChart {
	return &HelmChart{
		Name:     "",
		Version:  "",
		Registry: NewHelmRegistry(),
	}
}

// NewHelmRegistry returns the default Helm registry configuration
func NewHelmRegistry() *HelmRegistry {
	return &HelmRegistry{
		URL:      "",
		Type:     "HTTPS",
		Insecure: false,
		Auth:     NewHelmAuth(),
		TLS:      NewHelmTLS(),
	}
}

// NewHelmAuth returns the default Helm auth configuration
func NewHelmAuth() *HelmAuth {
	return &HelmAuth{
		Username: "",
		Password: "",
		Token:    "",
	}
}

// NewHelmTLS returns the default Helm TLS configuration
func NewHelmTLS() *HelmTLS {
	return &HelmTLS{
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
	// Validate chart name (no path traversal)
	if strings.Contains(h.Chart.Name, "..") || strings.Contains(h.Chart.Name, "/") {
		return fmt.Errorf("invalid chart name: contains invalid characters")
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
	// Validate URL
	if _, err := url.Parse(h.Chart.Registry.URL); err != nil {
		return fmt.Errorf("invalid registry URL: %w", err)
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
				cfg.Helm.Chart.Registry.Auth = NewHelmAuth()
			}
			if cfg.Helm.Chart.Registry.TLS == nil {
				cfg.Helm.Chart.Registry.TLS = NewHelmTLS()
			}
		}
	}

	return cfg, nil
}
