package cmd

import (
	"debug/buildinfo"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version subcommand
var (
	// exit is used to terminate the process; override for testing
	exit = os.Exit
	// exePath returns the path of the current executable; override for testing
	exePath = os.Executable
	// readBuildInfo reads embedded build info; override for testing
	readBuildInfo = buildinfo.ReadFile
)

// GetBuildVersion returns the build version information from the embedded build info
// Returns "development" if the version cannot be determined
func GetBuildVersion() string {
	exe, err := exePath()
	if err != nil {
		return "development"
	}
	info, err := readBuildInfo(exe)
	if err != nil {
		return "development"
	}

	// If we have a proper version, use it
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}

	// Otherwise, try to use the VCS revision
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" && setting.Value != "" {
			// Return the first 8 characters of the commit hash
			if len(setting.Value) > 8 {
				return setting.Value[:8]
			}
			return setting.Value
		}
	}

	return "development"
}

func showVersion() {
	version := GetBuildVersion()
	fmt.Println(version)
	exit(0)
}

func NewVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  `Print version information`,
		Run: func(cmd *cobra.Command, args []string) {
			showVersion()
		},
	}
	return cmd
}
