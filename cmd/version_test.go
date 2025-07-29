package cmd

import (
	"bytes"
	"debug/buildinfo"
	"errors"
	"io"
	"os"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBuildVersion(t *testing.T) {
	// Save original functions
	origExePath := exePath
	origReadBuildInfo := readBuildInfo
	defer func() {
		exePath = origExePath
		readBuildInfo = origReadBuildInfo
	}()

	tests := []struct {
		name          string
		setupFunc     func()
		expectedValue string
	}{
		{
			name: "exe path error returns development",
			setupFunc: func() {
				exePath = func() (string, error) {
					return "", errors.New("exe path error")
				}
			},
			expectedValue: "development",
		},
		{
			name: "build info error returns development",
			setupFunc: func() {
				exePath = func() (string, error) {
					return "/path/to/exe", nil
				}
				readBuildInfo = func(string) (*buildinfo.BuildInfo, error) {
					return nil, errors.New("build info error")
				}
			},
			expectedValue: "development",
		},
		{
			name: "proper version from Main.Version",
			setupFunc: func() {
				exePath = func() (string, error) {
					return "/path/to/exe", nil
				}
				readBuildInfo = func(string) (*buildinfo.BuildInfo, error) {
					return &buildinfo.BuildInfo{
						Main: debug.Module{
							Version: "v1.2.3",
						},
					}, nil
				}
			},
			expectedValue: "v1.2.3",
		},
		{
			name: "devel version falls back to vcs revision",
			setupFunc: func() {
				exePath = func() (string, error) {
					return "/path/to/exe", nil
				}
				readBuildInfo = func(string) (*buildinfo.BuildInfo, error) {
					return &buildinfo.BuildInfo{
						Main: debug.Module{
							Version: "(devel)",
						},
						Settings: []debug.BuildSetting{
							{Key: "vcs.revision", Value: "1234567890abcdef"},
						},
					}, nil
				}
			},
			expectedValue: "12345678",
		},
		{
			name: "short vcs revision",
			setupFunc: func() {
				exePath = func() (string, error) {
					return "/path/to/exe", nil
				}
				readBuildInfo = func(string) (*buildinfo.BuildInfo, error) {
					return &buildinfo.BuildInfo{
						Main: debug.Module{
							Version: "",
						},
						Settings: []debug.BuildSetting{
							{Key: "vcs.revision", Value: "abc123"},
						},
					}, nil
				}
			},
			expectedValue: "abc123",
		},
		{
			name: "no version or vcs revision",
			setupFunc: func() {
				exePath = func() (string, error) {
					return "/path/to/exe", nil
				}
				readBuildInfo = func(string) (*buildinfo.BuildInfo, error) {
					return &buildinfo.BuildInfo{
						Main: debug.Module{
							Version: "",
						},
						Settings: []debug.BuildSetting{},
					}, nil
				}
			},
			expectedValue: "development",
		},
		{
			name: "multiple settings with vcs revision",
			setupFunc: func() {
				exePath = func() (string, error) {
					return "/path/to/exe", nil
				}
				readBuildInfo = func(string) (*buildinfo.BuildInfo, error) {
					return &buildinfo.BuildInfo{
						Main: debug.Module{
							Version: "",
						},
						Settings: []debug.BuildSetting{
							{Key: "GOOS", Value: "linux"},
							{Key: "GOARCH", Value: "amd64"},
							{Key: "vcs.revision", Value: "fedcba0987654321"},
							{Key: "vcs.time", Value: "2023-01-01T00:00:00Z"},
						},
					}, nil
				}
			},
			expectedValue: "fedcba09",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFunc()
			result := GetBuildVersion()
			assert.Equal(t, tt.expectedValue, result)
		})
	}
}

func TestShowVersion(t *testing.T) {
	// Save original functions
	origExit := exit
	origExePath := exePath
	origReadBuildInfo := readBuildInfo
	defer func() {
		exit = origExit
		exePath = origExePath
		readBuildInfo = origReadBuildInfo
	}()

	// Capture exit code
	var exitCode int
	exit = func(code int) {
		exitCode = code
	}

	// Set up build info
	exePath = func() (string, error) {
		return "/path/to/exe", nil
	}
	readBuildInfo = func(string) (*buildinfo.BuildInfo, error) {
		return &buildinfo.BuildInfo{
			Main: debug.Module{
				Version: "v1.0.0",
			},
		}, nil
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	showVersion()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	assert.Equal(t, 0, exitCode)
	assert.Contains(t, buf.String(), "v1.0.0")
}

func TestNewVersionCmd(t *testing.T) {
	cmd := NewVersionCmd()

	assert.NotNil(t, cmd)
	assert.Equal(t, "version", cmd.Use)
	assert.Equal(t, "Print version information", cmd.Short)
	assert.NotNil(t, cmd.Run)
}

func TestVersionCmdExecution(t *testing.T) {
	// Save original functions
	origExit := exit
	origExePath := exePath
	origReadBuildInfo := readBuildInfo
	defer func() {
		exit = origExit
		exePath = origExePath
		readBuildInfo = origReadBuildInfo
	}()

	// Override exit to prevent test termination
	exitCalled := false
	exit = func(code int) {
		exitCalled = true
	}

	// Set up build info
	exePath = func() (string, error) {
		return "/path/to/exe", nil
	}
	readBuildInfo = func(string) (*buildinfo.BuildInfo, error) {
		return &buildinfo.BuildInfo{
			Main: debug.Module{
				Version: "v2.5.0",
			},
		}, nil
	}

	cmd := NewVersionCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var output bytes.Buffer
	io.Copy(&output, r)

	assert.True(t, exitCalled)
	assert.Contains(t, output.String(), "v2.5.0")
}
