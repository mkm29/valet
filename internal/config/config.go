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

	return h.Chart.Validate()
}

// Validate validates the HelmChart configuration
func (c *HelmChart) Validate() error {
	if c == nil {
		return fmt.Errorf("helm chart configuration is nil")
	}

	// Validate chart name
	if c.Name == "" {
		return fmt.Errorf("helm chart name is required")
	}
	if err := validateChartName(c.Name); err != nil {
		return fmt.Errorf("invalid chart name: %w", err)
	}

	// Validate chart version
	if c.Version == "" {
		return fmt.Errorf("helm chart version is required")
	}
	if err := validateChartVersion(c.Version); err != nil {
		return fmt.Errorf("invalid chart version: %w", err)
	}

	// Validate registry
	if c.Registry == nil {
		return fmt.Errorf("helm registry configuration is required")
	}
	if err := c.Registry.Validate(); err != nil {
		return fmt.Errorf("invalid registry configuration: %w", err)
	}

	return nil
}

// Validate validates the HelmRegistry configuration
func (r *HelmRegistry) Validate() error {
	if r == nil {
		return fmt.Errorf("registry configuration is nil")
	}

	// Validate URL
	if r.URL == "" {
		return fmt.Errorf("registry URL is required")
	}
	parsedURL, err := url.Parse(r.URL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Validate registry type
	validTypes := map[string]bool{"HTTP": true, "HTTPS": true, "OCI": true}
	if !validTypes[r.Type] {
		return fmt.Errorf("invalid registry type: %s (must be HTTP, HTTPS, or OCI)", r.Type)
	}

	// Additional URL scheme validation based on type
	switch r.Type {
	case "HTTP":
		if parsedURL.Scheme != "http" {
			return fmt.Errorf("HTTP registry type requires http:// URL scheme")
		}
	case "HTTPS":
		if parsedURL.Scheme != "https" {
			return fmt.Errorf("HTTPS registry type requires https:// URL scheme")
		}
	case "OCI":
		if parsedURL.Scheme != "oci" && parsedURL.Scheme != "https" {
			return fmt.Errorf("OCI registry type requires oci:// or https:// URL scheme")
		}
	}

	// Validate insecure flag consistency
	if r.Type == "HTTPS" && r.Insecure && r.TLS != nil && !r.TLS.InsecureSkipTLSVerify {
		return fmt.Errorf("conflicting TLS settings: insecure is true but InsecureSkipTLSVerify is false")
	}

	// Validate auth configuration
	if r.Auth != nil {
		if err := r.Auth.Validate(); err != nil {
			return fmt.Errorf("invalid auth configuration: %w", err)
		}
	}

	// Validate TLS configuration
	if r.TLS != nil {
		if err := r.TLS.Validate(); err != nil {
			return fmt.Errorf("invalid TLS configuration: %w", err)
		}
	}

	return nil
}

// Validate validates the HelmAuth configuration
func (a *HelmAuth) Validate() error {
	if a == nil {
		return nil // Auth is optional
	}

	// Check for conflicting auth methods
	authMethods := 0
	if a.Username != "" || a.Password != "" {
		authMethods++
		// Username and password must be provided together
		if a.Username == "" || a.Password == "" {
			return fmt.Errorf("both username and password must be provided for basic auth")
		}
	}
	if a.Token != "" {
		authMethods++
	}

	if authMethods > 1 {
		return fmt.Errorf("only one authentication method can be used at a time")
	}

	return nil
}

// Validate validates the HelmTLS configuration
func (t *HelmTLS) Validate() error {
	if t == nil {
		return nil // TLS is optional
	}

	// If cert file is provided, key file must also be provided
	if (t.CertFile != "" && t.KeyFile == "") || (t.CertFile == "" && t.KeyFile != "") {
		return fmt.Errorf("both cert file and key file must be provided for client TLS")
	}

	// Validate file paths exist if provided
	if t.CertFile != "" {
		if _, err := os.Stat(t.CertFile); err != nil {
			return fmt.Errorf("cert file not found: %w", err)
		}
	}
	if t.KeyFile != "" {
		if _, err := os.Stat(t.KeyFile); err != nil {
			return fmt.Errorf("key file not found: %w", err)
		}
	}
	if t.CaFile != "" {
		if _, err := os.Stat(t.CaFile); err != nil {
			return fmt.Errorf("CA file not found: %w", err)
		}
	}

	return nil
}

// validateChartName validates a Helm chart name
func validateChartName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	// Check for path traversal attempts
	if strings.Contains(name, "..") {
		return fmt.Errorf("name contains path traversal")
	}

	// Check for absolute paths
	if strings.HasPrefix(name, "/") || strings.HasPrefix(name, "\\") {
		return fmt.Errorf("name cannot be an absolute path")
	}

	// Check for invalid characters
	invalidChars := []string{"\\", ":", "*", "?", "\"", "<", ">", "|", "\n", "\r", "\t"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			return fmt.Errorf("name contains invalid character: %s", char)
		}
	}

	// Check length
	if len(name) > 255 {
		return fmt.Errorf("name is too long (max 255 characters)")
	}

	return nil
}

// validateChartVersion validates a Helm chart version
func validateChartVersion(version string) error {
	if version == "" {
		return fmt.Errorf("version cannot be empty")
	}

	// Check for invalid characters that could be used for injection
	invalidChars := []string{";", "&", "|", "$", "`", "(", ")", "{", "}", "[", "]", "<", ">", "\n", "\r", "\t"}
	for _, char := range invalidChars {
		if strings.Contains(version, char) {
			return fmt.Errorf("version contains invalid character: %s", char)
		}
	}

	// Check length
	if len(version) > 128 {
		return fmt.Errorf("version is too long (max 128 characters)")
	}

	// Basic semver pattern check (simplified)
	// This allows for versions like: 1.2.3, v1.2.3, 1.2.3-alpha, 1.2.3+build
	if !isValidVersion(version) {
		return fmt.Errorf("version does not appear to be a valid semantic version")
	}

	return nil
}

// isValidVersion performs a basic check if a version string looks valid
func isValidVersion(version string) bool {
	// Remove common prefixes
	v := strings.TrimPrefix(version, "v")
	v = strings.TrimPrefix(v, "V")

	// Very basic check - should start with a digit
	if len(v) == 0 || !isDigit(v[0]) {
		return false
	}

	// Should not have spaces
	if strings.Contains(v, " ") {
		return false
	}

	return true
}

// isDigit checks if a byte is a digit
func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
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

		// Validate Helm configuration
		if err := cfg.Helm.Validate(); err != nil {
			return nil, fmt.Errorf("invalid helm configuration: %w", err)
		}
	}

	// Validate telemetry configuration
	if err := cfg.Telemetry.Validate(); err != nil {
		return nil, fmt.Errorf("invalid telemetry configuration: %w", err)
	}

	return cfg, nil
}
