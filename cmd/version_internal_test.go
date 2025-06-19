package cmd

import (
	"bytes"
	"debug/buildinfo"
	"errors"
	"os"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShowVersion(t *testing.T) {
	tests := []struct {
		name           string
		exePathFunc    func() (string, error)
		buildInfoFunc  func(string) (*buildinfo.BuildInfo, error)
		expectedOutput string
		expectedExit   int
	}{
		{
			name: "successful version display with revision",
			exePathFunc: func() (string, error) {
				return "/path/to/exe", nil
			},
			buildInfoFunc: func(path string) (*buildinfo.BuildInfo, error) {
				return &buildinfo.BuildInfo{
					Main: debug.Module{
						Path:    "github.com/mkm29/valet",
						Version: "v1.0.0",
					},
					Settings: []debug.BuildSetting{
						{Key: "vcs.revision", Value: "abc123"},
						{Key: "vcs.time", Value: "2024-01-01"},
					},
				}, nil
			},
			expectedOutput: "github.com/mkm29/valet@v1.0.0 (commit abc123)\n",
			expectedExit:   0,
		},
		{
			name: "successful version display without revision",
			exePathFunc: func() (string, error) {
				return "/path/to/exe", nil
			},
			buildInfoFunc: func(path string) (*buildinfo.BuildInfo, error) {
				return &buildinfo.BuildInfo{
					Main: debug.Module{
						Path:    "github.com/mkm29/valet",
						Version: "v1.0.0",
					},
					Settings: []debug.BuildSetting{
						{Key: "vcs.time", Value: "2024-01-01"},
					},
				}, nil
			},
			expectedOutput: "github.com/mkm29/valet@v1.0.0\n",
			expectedExit:   0,
		},
		{
			name: "error getting executable path",
			exePathFunc: func() (string, error) {
				return "", errors.New("executable not found")
			},
			buildInfoFunc: func(path string) (*buildinfo.BuildInfo, error) {
				return nil, nil
			},
			expectedOutput: "error retrieving executable path: executable not found\n",
			expectedExit:   1,
		},
		{
			name: "error reading build info",
			exePathFunc: func() (string, error) {
				return "/path/to/exe", nil
			},
			buildInfoFunc: func(path string) (*buildinfo.BuildInfo, error) {
				return nil, errors.New("build info not available")
			},
			expectedOutput: "error reading build info: build info not available\n",
			expectedExit:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original functions
			originalExit := exit
			originalExePath := exePath
			originalReadBuildInfo := readBuildInfo

			// Capture exit code
			var exitCode int
			exit = func(code int) {
				exitCode = code
			}

			// Override functions
			exePath = tt.exePathFunc
			readBuildInfo = tt.buildInfoFunc

			// Capture output
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stdout = w
			os.Stderr = w

			// Run the function
			showVersion()

			// Restore output
			w.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr

			// Read captured output
			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			// Restore original functions
			exit = originalExit
			exePath = originalExePath
			readBuildInfo = originalReadBuildInfo

			// Assert
			assert.Equal(t, tt.expectedOutput, output)
			assert.Equal(t, tt.expectedExit, exitCode)
		})
	}
}

func TestVersionCommand_Integration(t *testing.T) {
	// Save original functions
	originalExit := exit
	originalExePath := exePath
	originalReadBuildInfo := readBuildInfo

	// Prevent actual exit
	var exitCalled bool
	exit = func(code int) {
		exitCalled = true
	}

	// Mock successful execution
	exePath = func() (string, error) {
		return "/test/exe", nil
	}
	readBuildInfo = func(path string) (*buildinfo.BuildInfo, error) {
		return &buildinfo.BuildInfo{
			Main: debug.Module{
				Path:    "github.com/mkm29/valet",
				Version: "v1.0.0",
			},
		}, nil
	}

	// Create and execute command
	cmd := NewVersionCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	err := cmd.Execute()

	// Restore original functions
	exit = originalExit
	exePath = originalExePath
	readBuildInfo = originalReadBuildInfo

	assert.NoError(t, err)
	assert.True(t, exitCalled)
}