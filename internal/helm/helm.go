package helm

// Determine if a remote chart contains a values.schema.json file

import (
	"fmt"
	"os"

	"github.com/mkm29/valet/internal/config"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/getter"
)

// Helm provides functionality for working with Helm charts
type Helm struct {
	logger *zap.Logger
	debug  bool
}

// HelmOptions configures a Helm instance
type HelmOptions struct {
	Debug  bool
	Logger *zap.Logger
}

// NewHelm creates a new Helm instance with options
func NewHelm(opts HelmOptions) *Helm {
	logger := opts.Logger
	if logger == nil {
		logger = zap.L().Named("helm")
	}

	return &Helm{
		logger: logger,
		debug:  opts.Debug,
	}
}

// NewHelmWithDebug creates a new Helm instance with just debug flag (convenience function)
func NewHelmWithDebug(debug bool) *Helm {
	return NewHelm(HelmOptions{
		Debug: debug,
	})
}

// GetOptions builds getter options from a HelmChart configuration
func (h *Helm) GetOptions(c *config.HelmChart) []getter.Option {
	var getterOpts []getter.Option

	if c.Registry.Type == "HTTP" {
		getterOpts = append(getterOpts, getter.WithPlainHTTP(true))
	}
	if c.Registry.Auth != nil && c.Registry.Auth.Username != "" && c.Registry.Auth.Password != "" {
		getterOpts = append(getterOpts, getter.WithBasicAuth(c.Registry.Auth.Username, c.Registry.Auth.Password))
	}
	if c.Registry.Insecure {
		getterOpts = append(getterOpts, getter.WithInsecureSkipVerifyTLS(true))
	}
	if c.Registry.TLS != nil && c.Registry.TLS.CertFile != "" && c.Registry.TLS.KeyFile != "" && c.Registry.TLS.CaFile != "" {
		getterOpts = append(getterOpts, getter.WithTLSClientConfig(c.Registry.TLS.CertFile, c.Registry.TLS.KeyFile, c.Registry.TLS.CaFile))
		getterOpts = append(getterOpts, getter.WithURL(c.Registry.URL))
	}

	return getterOpts
}

// HasSchema checks if a chart has a values.schema.json file
func (h *Helm) HasSchema(c *config.HelmChart) (bool, error) {
	url := fmt.Sprintf("%s/%s-%s.tgz", c.Registry.URL, c.Name, c.Version)

	// 1. Download the chart archive
	var g getter.Getter
	var e error
	// Removed usage of getter.Options as it does not exist in the Helm getter package
	switch c.Registry.Type {
	case "HTTP":
		g, e = getter.NewHTTPGetter()
		if e != nil {
			return false, fmt.Errorf("failed to create HTTP getter: %w", e)
		}
	case "OCI":
		g, e = getter.NewOCIGetter()
		if e != nil {
			return false, fmt.Errorf("failed to create OCI getter: %w", e)
		}
	default:
		return false, fmt.Errorf("unsupported registry type: %s", c.Registry.Type)
	}

	getterOpts := h.GetOptions(c)
	provider, err := g.Get(url, getterOpts...)
	if err != nil {
		return false, fmt.Errorf("failed to get chart: %w", err)
	}

	chart, err := loader.LoadArchive(provider)
	if err != nil {
		return false, fmt.Errorf("failed to load chart: %w", err)
	}

	// Check if the chart has a values.schema.json file
	for _, file := range chart.Raw {
		if h.debug {
			h.logger.Debug("Checking file", zap.String("file", file.Name))
		}
		if file.Name == "values.schema.json" {
			if h.debug {
				h.logger.Debug("Chart has values.schema.json")
			}
			return true, nil
		}
	}

	if h.debug {
		zap.L().Debug("Chart does not have values.schema.json")
	}
	return false, nil
}

// DownloadSchema retrieves the values.schema.json file from the chart and saves to temporary file
func (h *Helm) DownloadSchema(c *config.HelmChart) (string, error) {
	hasSchema, err := h.HasSchema(c)
	if err != nil {
		return "", fmt.Errorf("error checking for schema: %w", err)
	}
	if !hasSchema {
		// TODO: generate a schema
		return "", fmt.Errorf("chart does not have values.schema.json")
	}

	url := fmt.Sprintf("%s/%s-%s.tgz", c.Registry.URL, c.Name, c.Version)

	// 1. Download the chart archive
	var g getter.Getter
	var e error
	switch c.Registry.Type {
	case "HTTP":
		g, e = getter.NewHTTPGetter()
		if e != nil {
			return "", fmt.Errorf("failed to create HTTP getter: %w", e)
		}
	case "OCI":
		g, e = getter.NewOCIGetter()
		if e != nil {
			return "", fmt.Errorf("failed to create OCI getter: %w", e)
		}
	default:
		return "", fmt.Errorf("unsupported registry type: %s", c.Registry.Type)
	}

	getterOpts := h.GetOptions(c)
	provider, err := g.Get(url, getterOpts...)
	if err != nil {
		return "", fmt.Errorf("failed to get chart: %w", err)
	}

	chart, err := loader.LoadArchive(provider)
	if err != nil {
		return "", fmt.Errorf("failed to load chart: %w", err)
	}

	for _, file := range chart.Raw {
		if file.Name == "values.schema.json" {
			if h.debug {
				h.logger.Debug("Found values.schema.json in chart")
			}
			// write the schema to a temporary file
			tmp, err := os.CreateTemp("", "values.schema.json")
			if err != nil {
				return "", fmt.Errorf("failed to create temporary file: %w", err)
			}
			defer tmp.Close()
			if _, err := tmp.Write(file.Data); err != nil {
				return "", fmt.Errorf("failed to write to temporary file: %w", err)
			}
			if h.debug {
				h.logger.Debug("Schema saved to temporary file", zap.String("path", tmp.Name()))
			}
			// return the path to the temporary file
			// or return the schema as a string
			return tmp.Name(), nil
		}
	}

	return "", fmt.Errorf("values.schema.json not found in chart")
}
