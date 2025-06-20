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

	return cfg, nil
}
