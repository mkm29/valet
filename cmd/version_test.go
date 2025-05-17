package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"debug/buildinfo"
	debugpkg "runtime/debug"
)

func TestShowVersion_Success(t *testing.T) {
	// Backup and restore
	oldExit := exit
	oldRead := readBuildInfo
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		exit = oldExit
		readBuildInfo = oldRead
		os.Stdout = oldStdout
	}()
	// Stub build info
	readBuildInfo = func(path string) (*buildinfo.BuildInfo, error) {
		return &buildinfo.BuildInfo{
			Main:     debugpkg.Module{Path: "mod/path", Version: "vX.Y.Z"},
			Settings: []debugpkg.BuildSetting{{Key: "vcs.revision", Value: "abcdef"}},
		}, nil
	}
	// Capture exit code via panic
	exit = func(code int) { panic(code) }
	// Call showVersion and capture output
	var output bytes.Buffer
	done := make(chan struct{})
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				// Expect exit(0)
				if code, ok := rec.(int); !ok || code != 0 {
					t.Errorf("unexpected exit code: %v", rec)
				}
			} else {
				t.Error("expected exit(0) panic")
			}
			w.Close()
			io.Copy(&output, r)
			close(done)
		}()
		showVersion()
	}()
	<-done
	// Verify output
	expected := "mod/path@vX.Y.Z (commit abcdef)\n"
	if output.String() != expected {
		t.Errorf("expected %q, got %q", expected, output.String())
	}
}

func TestNewVersionCmd(t *testing.T) {
	cmd := NewVersionCmd()
	if cmd.Use != "version" {
		t.Errorf("expected Use 'version', got '%s'", cmd.Use)
	}
	if cmd.Short != "Print version information" {
		t.Errorf("expected Short 'Print version information', got '%s'", cmd.Short)
	}
	if cmd.Long == "" {
		t.Error("expected non-empty Long description")
	}
}

// TestShowVersion_ExeError simulates os.Executable failure
func TestShowVersion_ExeError(t *testing.T) {
	// Backup and restore
	oldExit := exit
	oldExe := exePath
	oldStdErr := os.Stderr
	defer func() {
		exit = oldExit
		exePath = oldExe
		os.Stderr = oldStdErr
	}()
	// Stub exePath to error
	exePath = func() (string, error) { return "", os.ErrPermission }
	// Capture stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	// Capture exit code via panic
	exit = func(code int) { panic(code) }
	var out bytes.Buffer
	// Run showVersion
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				if code, ok := rec.(int); !ok || code != 1 {
					t.Errorf("expected exit code=1, got %v", rec)
				}
			} else {
				t.Errorf("expected exit panic")
			}
			w.Close()
			io.Copy(&out, r)
		}()
		showVersion()
	}()
	if !bytes.Contains(out.Bytes(), []byte("error retrieving executable path")) {
		t.Errorf("expected exePath error message, got %q", out.String())
	}
}

// TestShowVersion_BuildInfoError simulates build info read failure
func TestShowVersion_BuildInfoError(t *testing.T) {
	oldExit := exit
	oldRead := readBuildInfo
	oldExe := exePath
	oldStdErr := os.Stderr
	defer func() {
		exit = oldExit
		readBuildInfo = oldRead
		exePath = oldExe
		os.Stderr = oldStdErr
	}()
	// Stub exePath to succeed
	exePath = func() (string, error) { return "/bin/true", nil }
	// Stub readBuildInfo to error
	readBuildInfo = func(path string) (*buildinfo.BuildInfo, error) { return nil, os.ErrInvalid }
	r, w, _ := os.Pipe()
	os.Stderr = w
	exit = func(code int) { panic(code) }
	var out bytes.Buffer
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				if code, ok := rec.(int); !ok || code != 1 {
					t.Errorf("expected exit code=1, got %v", rec)
				}
			} else {
				t.Errorf("expected exit panic")
			}
			w.Close()
			io.Copy(&out, r)
		}()
		showVersion()
	}()
	if !bytes.Contains(out.Bytes(), []byte("error reading build info")) {
		t.Errorf("expected build info error message, got %q", out.String())
	}
}
