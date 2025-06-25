package tests

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/mkm29/valet/internal/config"
	"github.com/stretchr/testify/suite"
)

type ConfigValidationTestSuite struct {
	ValetTestSuite
	tempDir string
}

func (suite *ConfigValidationTestSuite) SetupSuite() {
	suite.tempDir = suite.T().TempDir()
}

func (suite *ConfigValidationTestSuite) TestHelmConfig_Validate() {
	tests := []struct {
		name    string
		config  *config.HelmConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config is valid",
			config:  nil,
			wantErr: false,
		},
		{
			name:    "nil chart is valid",
			config:  &config.HelmConfig{Chart: nil},
			wantErr: false,
		},
		{
			name: "valid config",
			config: &config.HelmConfig{
				Chart: &config.HelmChart{
					Name:    "my-chart",
					Version: "1.2.3",
					Registry: &config.HelmRegistry{
						URL:  "https://example.com",
						Type: "HTTPS",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := tt.config.Validate()
			if tt.wantErr {
				suite.Error(err)
				if tt.errMsg != "" {
					suite.Contains(err.Error(), tt.errMsg)
				}
			} else {
				suite.NoError(err)
			}
		})
	}
}

func (suite *ConfigValidationTestSuite) TestHelmChart_Validate() {
	tests := []struct {
		name    string
		chart   *config.HelmChart
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil chart",
			chart:   nil,
			wantErr: true,
			errMsg:  "helm chart configuration is nil",
		},
		{
			name: "empty name",
			chart: &config.HelmChart{
				Name:     "",
				Version:  "1.0.0",
				Registry: &config.HelmRegistry{URL: "https://example.com", Type: "HTTPS"},
			},
			wantErr: true,
			errMsg:  "helm chart name is required",
		},
		{
			name: "invalid name with path traversal",
			chart: &config.HelmChart{
				Name:     "../../../etc/passwd",
				Version:  "1.0.0",
				Registry: &config.HelmRegistry{URL: "https://example.com", Type: "HTTPS"},
			},
			wantErr: true,
			errMsg:  "name contains path traversal",
		},
		{
			name: "invalid name with absolute path",
			chart: &config.HelmChart{
				Name:     "/etc/passwd",
				Version:  "1.0.0",
				Registry: &config.HelmRegistry{URL: "https://example.com", Type: "HTTPS"},
			},
			wantErr: true,
			errMsg:  "name cannot be an absolute path",
		},
		{
			name: "invalid name with special chars",
			chart: &config.HelmChart{
				Name:     "my-chart|echo test",
				Version:  "1.0.0",
				Registry: &config.HelmRegistry{URL: "https://example.com", Type: "HTTPS"},
			},
			wantErr: true,
			errMsg:  "name contains invalid character",
		},
		{
			name: "name too long",
			chart: &config.HelmChart{
				Name:     string(make([]byte, 256)),
				Version:  "1.0.0",
				Registry: &config.HelmRegistry{URL: "https://example.com", Type: "HTTPS"},
			},
			wantErr: true,
			errMsg:  "name is too long",
		},
		{
			name: "empty version",
			chart: &config.HelmChart{
				Name:     "my-chart",
				Version:  "",
				Registry: &config.HelmRegistry{URL: "https://example.com", Type: "HTTPS"},
			},
			wantErr: true,
			errMsg:  "helm chart version is required",
		},
		{
			name: "invalid version with command injection",
			chart: &config.HelmChart{
				Name:     "my-chart",
				Version:  "1.0.0; rm -rf /",
				Registry: &config.HelmRegistry{URL: "https://example.com", Type: "HTTPS"},
			},
			wantErr: true,
			errMsg:  "version contains invalid character",
		},
		{
			name: "version too long",
			chart: &config.HelmChart{
				Name:     "my-chart",
				Version:  string(make([]byte, 129)),
				Registry: &config.HelmRegistry{URL: "https://example.com", Type: "HTTPS"},
			},
			wantErr: true,
			errMsg:  "version is too long",
		},
		{
			name: "invalid version format",
			chart: &config.HelmChart{
				Name:     "my-chart",
				Version:  "not-a-version",
				Registry: &config.HelmRegistry{URL: "https://example.com", Type: "HTTPS"},
			},
			wantErr: true,
			errMsg:  "version does not appear to be a valid semantic version",
		},
		{
			name: "nil registry",
			chart: &config.HelmChart{
				Name:     "my-chart",
				Version:  "1.0.0",
				Registry: nil,
			},
			wantErr: true,
			errMsg:  "helm registry configuration is required",
		},
		{
			name: "valid semantic versions",
			chart: &config.HelmChart{
				Name:     "my-chart",
				Version:  "v1.2.3-alpha.1+build.123",
				Registry: &config.HelmRegistry{URL: "https://example.com", Type: "HTTPS"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := tt.chart.Validate()
			if tt.wantErr {
				suite.Error(err)
				if tt.errMsg != "" {
					suite.Contains(err.Error(), tt.errMsg)
				}
			} else {
				suite.NoError(err)
			}
		})
	}
}

func (suite *ConfigValidationTestSuite) TestHelmRegistry_Validate() {
	tests := []struct {
		name    string
		reg     *config.HelmRegistry
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil registry",
			reg:     nil,
			wantErr: true,
			errMsg:  "registry configuration is nil",
		},
		{
			name: "empty URL",
			reg: &config.HelmRegistry{
				URL:  "",
				Type: "HTTPS",
			},
			wantErr: true,
			errMsg:  "registry URL is required",
		},
		{
			name: "invalid URL",
			reg: &config.HelmRegistry{
				URL:  "not a url",
				Type: "HTTPS",
			},
			wantErr: true,
			errMsg:  "HTTPS registry type requires https:// URL scheme",
		},
		{
			name: "invalid registry type",
			reg: &config.HelmRegistry{
				URL:  "https://example.com",
				Type: "FTP",
			},
			wantErr: true,
			errMsg:  "invalid registry type",
		},
		{
			name: "HTTP type with HTTPS URL",
			reg: &config.HelmRegistry{
				URL:  "https://example.com",
				Type: "HTTP",
			},
			wantErr: true,
			errMsg:  "HTTP registry type requires http:// URL scheme",
		},
		{
			name: "HTTPS type with HTTP URL",
			reg: &config.HelmRegistry{
				URL:  "http://example.com",
				Type: "HTTPS",
			},
			wantErr: true,
			errMsg:  "HTTPS registry type requires https:// URL scheme",
		},
		{
			name: "OCI type with invalid scheme",
			reg: &config.HelmRegistry{
				URL:  "http://example.com",
				Type: "OCI",
			},
			wantErr: true,
			errMsg:  "OCI registry type requires oci:// or https:// URL scheme",
		},
		{
			name: "conflicting TLS settings",
			reg: &config.HelmRegistry{
				URL:      "https://example.com",
				Type:     "HTTPS",
				Insecure: true,
				TLS: &config.HelmTLS{
					InsecureSkipTLSVerify: false,
				},
			},
			wantErr: true,
			errMsg:  "conflicting TLS settings",
		},
		{
			name: "valid OCI registry with oci scheme",
			reg: &config.HelmRegistry{
				URL:  "oci://example.com/charts",
				Type: "OCI",
			},
			wantErr: false,
		},
		{
			name: "valid OCI registry with https scheme",
			reg: &config.HelmRegistry{
				URL:  "https://example.com/charts",
				Type: "OCI",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := tt.reg.Validate()
			if tt.wantErr {
				suite.Error(err)
				if tt.errMsg != "" {
					suite.Contains(err.Error(), tt.errMsg)
				}
			} else {
				suite.NoError(err)
			}
		})
	}
}

func (suite *ConfigValidationTestSuite) TestHelmAuth_Validate() {
	tests := []struct {
		name    string
		auth    *config.HelmAuth
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil auth is valid",
			auth:    nil,
			wantErr: false,
		},
		{
			name:    "empty auth is valid",
			auth:    &config.HelmAuth{},
			wantErr: false,
		},
		{
			name: "username without password",
			auth: &config.HelmAuth{
				Username: "user",
				Password: "",
			},
			wantErr: true,
			errMsg:  "both username and password must be provided",
		},
		{
			name: "password without username",
			auth: &config.HelmAuth{
				Username: "",
				Password: "pass",
			},
			wantErr: true,
			errMsg:  "both username and password must be provided",
		},
		{
			name: "valid basic auth",
			auth: &config.HelmAuth{
				Username: "user",
				Password: "pass",
			},
			wantErr: false,
		},
		{
			name: "valid token auth",
			auth: &config.HelmAuth{
				Token: "my-token",
			},
			wantErr: false,
		},
		{
			name: "conflicting auth methods",
			auth: &config.HelmAuth{
				Username: "user",
				Password: "pass",
				Token:    "token",
			},
			wantErr: true,
			errMsg:  "only one authentication method can be used",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := tt.auth.Validate()
			if tt.wantErr {
				suite.Error(err)
				if tt.errMsg != "" {
					suite.Contains(err.Error(), tt.errMsg)
				}
			} else {
				suite.NoError(err)
			}
		})
	}
}

func (suite *ConfigValidationTestSuite) TestHelmTLS_Validate() {
	// Create temporary files for testing
	certFile := filepath.Join(suite.tempDir, "cert.pem")
	keyFile := filepath.Join(suite.tempDir, "key.pem")
	caFile := filepath.Join(suite.tempDir, "ca.pem")

	suite.NoError(os.WriteFile(certFile, []byte("cert"), 0600))
	suite.NoError(os.WriteFile(keyFile, []byte("key"), 0600))
	suite.NoError(os.WriteFile(caFile, []byte("ca"), 0600))

	tests := []struct {
		name    string
		tls     *config.HelmTLS
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil TLS is valid",
			tls:     nil,
			wantErr: false,
		},
		{
			name:    "empty TLS is valid",
			tls:     &config.HelmTLS{},
			wantErr: false,
		},
		{
			name: "cert without key",
			tls: &config.HelmTLS{
				CertFile: certFile,
				KeyFile:  "",
			},
			wantErr: true,
			errMsg:  "both cert file and key file must be provided",
		},
		{
			name: "key without cert",
			tls: &config.HelmTLS{
				CertFile: "",
				KeyFile:  keyFile,
			},
			wantErr: true,
			errMsg:  "both cert file and key file must be provided",
		},
		{
			name: "valid client TLS",
			tls: &config.HelmTLS{
				CertFile: certFile,
				KeyFile:  keyFile,
				CaFile:   caFile,
			},
			wantErr: false,
		},
		{
			name: "non-existent cert file",
			tls: &config.HelmTLS{
				CertFile: "/non/existent/cert.pem",
				KeyFile:  keyFile,
			},
			wantErr: true,
			errMsg:  "cert file not found",
		},
		{
			name: "non-existent key file",
			tls: &config.HelmTLS{
				CertFile: certFile,
				KeyFile:  "/non/existent/key.pem",
			},
			wantErr: true,
			errMsg:  "key file not found",
		},
		{
			name: "non-existent CA file",
			tls: &config.HelmTLS{
				CaFile: "/non/existent/ca.pem",
			},
			wantErr: true,
			errMsg:  "CA file not found",
		},
		{
			name: "insecure skip TLS verify",
			tls: &config.HelmTLS{
				InsecureSkipTLSVerify: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := tt.tls.Validate()
			if tt.wantErr {
				suite.Error(err)
				if tt.errMsg != "" {
					suite.Contains(err.Error(), tt.errMsg)
				}
			} else {
				suite.NoError(err)
			}
		})
	}
}

func (suite *ConfigValidationTestSuite) TestLoadConfig_WithValidation() {
	suite.Run("invalid helm config", func() {
		configFile := filepath.Join(suite.tempDir, "invalid-helm.yaml")
		content := `
helm:
  chart:
    name: "../../../etc/passwd"
    version: "1.0.0"
    registry:
      url: "https://example.com"
      type: "HTTPS"
`
		suite.NoError(os.WriteFile(configFile, []byte(content), 0644))

		_, err := config.LoadConfig(configFile)
		suite.Error(err)
		suite.Contains(err.Error(), "invalid helm configuration")
		suite.Contains(err.Error(), "path traversal")
	})

	suite.Run("invalid telemetry config", func() {
		configFile := filepath.Join(suite.tempDir, "invalid-telemetry.yaml")
		content := `
telemetry:
  enabled: true
  exporterType: "invalid"
  sampleRate: 2.0
`
		suite.NoError(os.WriteFile(configFile, []byte(content), 0644))

		_, err := config.LoadConfig(configFile)
		suite.Error(err)
		suite.Contains(err.Error(), "invalid telemetry configuration")
	})

	suite.Run("valid config", func() {
		configFile := filepath.Join(suite.tempDir, "valid.yaml")
		content := `
debug: true
telemetry:
  enabled: true
  exporterType: "otlp"
  sampleRate: 0.5
helm:
  chart:
    name: "my-chart"
    version: "v1.2.3"
    registry:
      url: "https://charts.example.com"
      type: "HTTPS"
`
		suite.NoError(os.WriteFile(configFile, []byte(content), 0644))

		cfg, err := config.LoadConfig(configFile)
		suite.NoError(err)
		suite.NotNil(cfg)
		suite.Equal(slog.LevelDebug, cfg.LogLevel.Level)
		suite.Equal("my-chart", cfg.Helm.Chart.Name)
	})
}

func TestConfigValidationSuite(t *testing.T) {
	suite.Run(t, new(ConfigValidationTestSuite))
}
