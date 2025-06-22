package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestHelmConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *HelmConfig
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
			config:  &HelmConfig{Chart: nil},
			wantErr: false,
		},
		{
			name: "valid config",
			config: &HelmConfig{
				Chart: &HelmChart{
					Name:    "my-chart",
					Version: "1.2.3",
					Registry: &HelmRegistry{
						URL:  "https://example.com",
						Type: "HTTPS",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHelmChart_Validate(t *testing.T) {
	tests := []struct {
		name    string
		chart   *HelmChart
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
			chart: &HelmChart{
				Name:     "",
				Version:  "1.0.0",
				Registry: &HelmRegistry{URL: "https://example.com", Type: "HTTPS"},
			},
			wantErr: true,
			errMsg:  "helm chart name is required",
		},
		{
			name: "invalid name with path traversal",
			chart: &HelmChart{
				Name:     "../../../etc/passwd",
				Version:  "1.0.0",
				Registry: &HelmRegistry{URL: "https://example.com", Type: "HTTPS"},
			},
			wantErr: true,
			errMsg:  "name contains path traversal",
		},
		{
			name: "invalid name with absolute path",
			chart: &HelmChart{
				Name:     "/etc/passwd",
				Version:  "1.0.0",
				Registry: &HelmRegistry{URL: "https://example.com", Type: "HTTPS"},
			},
			wantErr: true,
			errMsg:  "name cannot be an absolute path",
		},
		{
			name: "invalid name with special chars",
			chart: &HelmChart{
				Name:     "my-chart|echo test",
				Version:  "1.0.0",
				Registry: &HelmRegistry{URL: "https://example.com", Type: "HTTPS"},
			},
			wantErr: true,
			errMsg:  "name contains invalid character",
		},
		{
			name: "name too long",
			chart: &HelmChart{
				Name:     string(make([]byte, 256)),
				Version:  "1.0.0",
				Registry: &HelmRegistry{URL: "https://example.com", Type: "HTTPS"},
			},
			wantErr: true,
			errMsg:  "name is too long",
		},
		{
			name: "empty version",
			chart: &HelmChart{
				Name:     "my-chart",
				Version:  "",
				Registry: &HelmRegistry{URL: "https://example.com", Type: "HTTPS"},
			},
			wantErr: true,
			errMsg:  "helm chart version is required",
		},
		{
			name: "invalid version with command injection",
			chart: &HelmChart{
				Name:     "my-chart",
				Version:  "1.0.0; rm -rf /",
				Registry: &HelmRegistry{URL: "https://example.com", Type: "HTTPS"},
			},
			wantErr: true,
			errMsg:  "version contains invalid character",
		},
		{
			name: "version too long",
			chart: &HelmChart{
				Name:     "my-chart",
				Version:  string(make([]byte, 129)),
				Registry: &HelmRegistry{URL: "https://example.com", Type: "HTTPS"},
			},
			wantErr: true,
			errMsg:  "version is too long",
		},
		{
			name: "invalid version format",
			chart: &HelmChart{
				Name:     "my-chart",
				Version:  "not-a-version",
				Registry: &HelmRegistry{URL: "https://example.com", Type: "HTTPS"},
			},
			wantErr: true,
			errMsg:  "version does not appear to be a valid semantic version",
		},
		{
			name: "nil registry",
			chart: &HelmChart{
				Name:     "my-chart",
				Version:  "1.0.0",
				Registry: nil,
			},
			wantErr: true,
			errMsg:  "helm registry configuration is required",
		},
		{
			name: "valid semantic versions",
			chart: &HelmChart{
				Name:     "my-chart",
				Version:  "v1.2.3-alpha.1+build.123",
				Registry: &HelmRegistry{URL: "https://example.com", Type: "HTTPS"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.chart.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHelmRegistry_Validate(t *testing.T) {
	tests := []struct {
		name    string
		reg     *HelmRegistry
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
			reg: &HelmRegistry{
				URL:  "",
				Type: "HTTPS",
			},
			wantErr: true,
			errMsg:  "registry URL is required",
		},
		{
			name: "invalid URL",
			reg: &HelmRegistry{
				URL:  "not a url",
				Type: "HTTPS",
			},
			wantErr: true,
			errMsg:  "HTTPS registry type requires https:// URL scheme",
		},
		{
			name: "invalid registry type",
			reg: &HelmRegistry{
				URL:  "https://example.com",
				Type: "FTP",
			},
			wantErr: true,
			errMsg:  "invalid registry type",
		},
		{
			name: "HTTP type with HTTPS URL",
			reg: &HelmRegistry{
				URL:  "https://example.com",
				Type: "HTTP",
			},
			wantErr: true,
			errMsg:  "HTTP registry type requires http:// URL scheme",
		},
		{
			name: "HTTPS type with HTTP URL",
			reg: &HelmRegistry{
				URL:  "http://example.com",
				Type: "HTTPS",
			},
			wantErr: true,
			errMsg:  "HTTPS registry type requires https:// URL scheme",
		},
		{
			name: "OCI type with invalid scheme",
			reg: &HelmRegistry{
				URL:  "http://example.com",
				Type: "OCI",
			},
			wantErr: true,
			errMsg:  "OCI registry type requires oci:// or https:// URL scheme",
		},
		{
			name: "conflicting TLS settings",
			reg: &HelmRegistry{
				URL:      "https://example.com",
				Type:     "HTTPS",
				Insecure: true,
				TLS: &HelmTLS{
					InsecureSkipTLSVerify: false,
				},
			},
			wantErr: true,
			errMsg:  "conflicting TLS settings",
		},
		{
			name: "valid OCI registry with oci scheme",
			reg: &HelmRegistry{
				URL:  "oci://example.com/charts",
				Type: "OCI",
			},
			wantErr: false,
		},
		{
			name: "valid OCI registry with https scheme",
			reg: &HelmRegistry{
				URL:  "https://example.com/charts",
				Type: "OCI",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.reg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHelmAuth_Validate(t *testing.T) {
	tests := []struct {
		name    string
		auth    *HelmAuth
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
			auth:    &HelmAuth{},
			wantErr: false,
		},
		{
			name: "username without password",
			auth: &HelmAuth{
				Username: "user",
				Password: "",
			},
			wantErr: true,
			errMsg:  "both username and password must be provided",
		},
		{
			name: "password without username",
			auth: &HelmAuth{
				Username: "",
				Password: "pass",
			},
			wantErr: true,
			errMsg:  "both username and password must be provided",
		},
		{
			name: "valid basic auth",
			auth: &HelmAuth{
				Username: "user",
				Password: "pass",
			},
			wantErr: false,
		},
		{
			name: "valid token auth",
			auth: &HelmAuth{
				Token: "my-token",
			},
			wantErr: false,
		},
		{
			name: "conflicting auth methods",
			auth: &HelmAuth{
				Username: "user",
				Password: "pass",
				Token:    "token",
			},
			wantErr: true,
			errMsg:  "only one authentication method can be used",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.auth.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHelmTLS_Validate(t *testing.T) {
	// Create temporary files for testing
	tempDir := t.TempDir()
	certFile := filepath.Join(tempDir, "cert.pem")
	keyFile := filepath.Join(tempDir, "key.pem")
	caFile := filepath.Join(tempDir, "ca.pem")

	require.NoError(t, os.WriteFile(certFile, []byte("cert"), 0600))
	require.NoError(t, os.WriteFile(keyFile, []byte("key"), 0600))
	require.NoError(t, os.WriteFile(caFile, []byte("ca"), 0600))

	tests := []struct {
		name    string
		tls     *HelmTLS
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
			tls:     &HelmTLS{},
			wantErr: false,
		},
		{
			name: "cert without key",
			tls: &HelmTLS{
				CertFile: certFile,
				KeyFile:  "",
			},
			wantErr: true,
			errMsg:  "both cert file and key file must be provided",
		},
		{
			name: "key without cert",
			tls: &HelmTLS{
				CertFile: "",
				KeyFile:  keyFile,
			},
			wantErr: true,
			errMsg:  "both cert file and key file must be provided",
		},
		{
			name: "valid client TLS",
			tls: &HelmTLS{
				CertFile: certFile,
				KeyFile:  keyFile,
				CaFile:   caFile,
			},
			wantErr: false,
		},
		{
			name: "non-existent cert file",
			tls: &HelmTLS{
				CertFile: "/non/existent/cert.pem",
				KeyFile:  keyFile,
			},
			wantErr: true,
			errMsg:  "cert file not found",
		},
		{
			name: "non-existent key file",
			tls: &HelmTLS{
				CertFile: certFile,
				KeyFile:  "/non/existent/key.pem",
			},
			wantErr: true,
			errMsg:  "key file not found",
		},
		{
			name: "non-existent CA file",
			tls: &HelmTLS{
				CaFile: "/non/existent/ca.pem",
			},
			wantErr: true,
			errMsg:  "CA file not found",
		},
		{
			name: "insecure skip TLS verify",
			tls: &HelmTLS{
				InsecureSkipTLSVerify: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tls.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateChartName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid name",
			input:   "my-chart",
			wantErr: false,
		},
		{
			name:    "valid name with numbers",
			input:   "chart-123",
			wantErr: false,
		},
		{
			name:    "empty name",
			input:   "",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "path traversal",
			input:   "../../../etc/passwd",
			wantErr: true,
			errMsg:  "path traversal",
		},
		{
			name:    "absolute path unix",
			input:   "/etc/passwd",
			wantErr: true,
			errMsg:  "absolute path",
		},
		{
			name:    "absolute path windows",
			input:   "\\windows\\system32",
			wantErr: true,
			errMsg:  "absolute path",
		},
		{
			name:    "contains backslash",
			input:   "my\\chart",
			wantErr: true,
			errMsg:  "invalid character: \\",
		},
		{
			name:    "contains pipe",
			input:   "my|chart",
			wantErr: true,
			errMsg:  "invalid character: |",
		},
		{
			name:    "contains newline",
			input:   "my\nchart",
			wantErr: true,
			errMsg:  "invalid character: \n",
		},
		{
			name:    "too long",
			input:   string(make([]byte, 256)),
			wantErr: true,
			errMsg:  "too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateChartName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateChartVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid semver",
			input:   "1.2.3",
			wantErr: false,
		},
		{
			name:    "valid semver with v prefix",
			input:   "v1.2.3",
			wantErr: false,
		},
		{
			name:    "valid semver with prerelease",
			input:   "1.2.3-alpha.1",
			wantErr: false,
		},
		{
			name:    "valid semver with metadata",
			input:   "1.2.3+build.123",
			wantErr: false,
		},
		{
			name:    "empty version",
			input:   "",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "command injection semicolon",
			input:   "1.0.0; rm -rf /",
			wantErr: true,
			errMsg:  "invalid character: ;",
		},
		{
			name:    "command injection pipe",
			input:   "1.0.0 | echo test",
			wantErr: true,
			errMsg:  "invalid character: |",
		},
		{
			name:    "contains backtick",
			input:   "1.0.0`echo test`",
			wantErr: true,
			errMsg:  "invalid character: `",
		},
		{
			name:    "too long",
			input:   string(make([]byte, 129)),
			wantErr: true,
			errMsg:  "too long",
		},
		{
			name:    "not starting with digit",
			input:   "alpha-1.2.3",
			wantErr: true,
			errMsg:  "not appear to be a valid semantic version",
		},
		{
			name:    "contains space",
			input:   "1.2.3 alpha",
			wantErr: true,
			errMsg:  "not appear to be a valid semantic version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateChartVersion(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoadConfig_WithValidation(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("invalid helm config", func(t *testing.T) {
		configFile := filepath.Join(tempDir, "invalid-helm.yaml")
		content := `
helm:
  chart:
    name: "../../../etc/passwd"
    version: "1.0.0"
    registry:
      url: "https://example.com"
      type: "HTTPS"
`
		require.NoError(t, os.WriteFile(configFile, []byte(content), 0644))

		_, err := LoadConfig(configFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid helm configuration")
		assert.Contains(t, err.Error(), "path traversal")
	})

	t.Run("invalid telemetry config", func(t *testing.T) {
		configFile := filepath.Join(tempDir, "invalid-telemetry.yaml")
		content := `
telemetry:
  enabled: true
  exporterType: "invalid"
  sampleRate: 2.0
`
		require.NoError(t, os.WriteFile(configFile, []byte(content), 0644))

		_, err := LoadConfig(configFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid telemetry configuration")
	})

	t.Run("valid config", func(t *testing.T) {
		configFile := filepath.Join(tempDir, "valid.yaml")
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
		require.NoError(t, os.WriteFile(configFile, []byte(content), 0644))

		cfg, err := LoadConfig(configFile)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, zapcore.DebugLevel, cfg.LogLevel.Level)
		assert.Equal(t, "my-chart", cfg.Helm.Chart.Name)
	})
}
