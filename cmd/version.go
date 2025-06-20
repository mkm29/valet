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

func showVersion() {
	exe, err := exePath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error retrieving executable path: %v\n", err)
		exit(1)
	}
	info, err := readBuildInfo(exe)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading build info: %v\n", err)
		exit(1)
	}
	// print main module path, version, and VCS revision if available
	revision := ""
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			revision = setting.Value
			break
		}
	}
	if revision != "" {
		fmt.Printf("%s@%s (commit %s)\n", info.Main.Path, info.Main.Version, revision)
	} else {
		fmt.Printf("%s@%s\n", info.Main.Path, info.Main.Version)
	}
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
