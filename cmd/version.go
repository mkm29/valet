package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)


// NewVersionCmdWithApp creates version command with dependency injection
func NewVersionCmdWithApp(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  `Print version information for the valet CLI.`,
		// Do not print usage on error; just show the error message
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use the refactored GetBuildVersion function
			version := GetBuildVersion()
			fmt.Println(version)
			return nil
		},
	}
	return cmd
}

