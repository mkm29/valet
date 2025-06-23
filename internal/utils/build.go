package utils

import (
	"fmt"
	"runtime/debug"
)

// GetBuildVersion returns the build version information
func GetBuildVersion() string {
	// Default version when running directly with `go run`
	version := "(unknown version)"
	commit := "(unknown commit)"

	// Read build info
	info, ok := debug.ReadBuildInfo()
	if ok {
		// Extract version from module
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}

		// Extract commit from VCS settings
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				commit = setting.Value
				break
			}
		}
	}

	// Format output
	if commit != "(unknown commit)" {
		return fmt.Sprintf("%s@%s (commit %s)", info.Main.Path, version, commit)
	}
	return fmt.Sprintf("%s@%s", info.Main.Path, version)
}
