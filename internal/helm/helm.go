package helm

// Determine if a remote chart contains a values.schema.json file

import (
	"fmt"
	"log"

	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/getter"
)

type Registry struct {
	URL  string
	Auth struct {
		Username string // Optional username for authentication
		Password string // Optional password for authentication
		Token    string // Optional authentication token for private registries
	} // Optional authentication token for private registries
	TLS struct {
		InsecureSkipTLSVerify bool   // Whether to skip TLS verification
		CertFile              string // Path to the client certificate file
		KeyFile               string // Path to the client key file
		CaFile                string // Path to the CA certificate file
	} // optional TLS configuration
	Insecure bool   // Whether to allow insecure connections
	Type     string // e.g., "HTTP", "HTTPS", "OCI"
}

type Chart struct {
	Registry *Registry
	Name     string
	Version  string
}

// ChartHasSchema checks if a chart has a values.schema.json file
func (c *Chart) HasSchema() (bool, error) {
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

	getterOpts := []getter.Option{}
	if c.Registry.Type == "HTTP" {
		getterOpts = append(getterOpts, getter.WithPlainHTTP(true))
	}
	if c.Registry.Auth.Username != "" && c.Registry.Auth.Password != "" {
		getterOpts = append(getterOpts, getter.WithBasicAuth(c.Registry.Auth.Username, c.Registry.Auth.Password))
	}
	// if c.Registry.Auth.Token != "" {
	// 	getterOpts = append(getterOpts, getter.WithToken(c.Registry.Auth.Token))
	// }
	if c.Registry.Insecure {
		getterOpts = append(getterOpts, getter.WithInsecureSkipVerifyTLS(true))
	}
	if c.Registry.TLS.CertFile != "" && c.Registry.TLS.KeyFile != "" && c.Registry.TLS.CaFile != "" {
		getterOpts = append(getterOpts, getter.WithTLSClientConfig(c.Registry.TLS.CertFile, c.Registry.TLS.KeyFile, c.Registry.TLS.CaFile))
		getterOpts = append(getterOpts, getter.WithURL(url))
	}
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
