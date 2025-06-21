package helm

// Determine if a remote chart contains a values.schema.json file

import (
	"fmt"
	"log"
	"os"

	"github.com/mkm29/valet/internal/config"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/getter"
)

// GetOptions builds getter options from a HelmChart configuration
func GetOptions(c *config.HelmChart) []getter.Option {
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
func HasSchema(c *config.HelmChart) (bool, error) {
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

	getterOpts := GetOptions(c)
	provider, err := g.Get(url, getterOpts...)
	if err != nil {
		return false, fmt.Errorf("failed to get chart: %w", err)
	}

	chart, err := loader.LoadArchive(provider)
	if err != nil {
		return false, fmt.Errorf("failed to load chart: %w", err)
	}

	// Check if the chart has a values.schema.json file
	for _, file := range chart.Files {
		if file.Name == "values.schema.json" {
			log.Println("Chart has values.schema.json")
			return true, nil
		}
	}

	log.Println("Chart does not have values.schema.json")
	return false, nil
}

// DownloadSchema retrieves the values.schema.json file from the chart and saves to temporary file
func DownloadSchema(c *config.HelmChart) (string, error) {
	hasSchema, err := HasSchema(c)
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

	getterOpts := GetOptions(c)
	provider, err := g.Get(url, getterOpts...)
	if err != nil {
		return "", fmt.Errorf("failed to get chart: %w", err)
	}

	chart, err := loader.LoadArchive(provider)
	if err != nil {
		return "", fmt.Errorf("failed to load chart: %w", err)
	}

	for _, file := range chart.Files {
		if file.Name == "values.schema.json" {
			log.Println("Found values.schema.json in chart")
			// write the schema to a temporary file
			tmp, err := os.CreateTemp("", "values.schema.json")
			if err != nil {
				return "", fmt.Errorf("failed to create temporary file: %w", err)
			}
			defer tmp.Close()
			if _, err := tmp.Write(file.Data); err != nil {
				return "", fmt.Errorf("failed to write to temporary file: %w", err)
			}
			log.Printf("Schema saved to temporary file: %s", tmp.Name())
			// return the path to the temporary file
			// or return the schema as a string
			return tmp.Name(), nil
		}
	}

	return "", fmt.Errorf("values.schema.json not found in chart")
}
