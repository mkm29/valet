package cmd

import (
	"fmt"
	"log"

	"github.com/mkm29/valet/internal/config"
	"github.com/spf13/cobra"
)

var cfg *config.Config

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "valet",
		Short: "JSON Schema Generator",
		Long:  `A JSON Schema Generator for Helm charts and other YAML files.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cfg == nil {
				c, err := initializeConfig(cmd)
				if err != nil {
					return err
				}
				cfg = c
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default action: generate schema
			ctx := cfg.Context
			if len(args) > 0 && args[0] != "" {
				ctx = args[0]
			}
			if ctx == "" {
				return cmd.Help()
			}
			msg, err := Generate(ctx, cfg.Overrides)
			if err != nil {
				return err
			}
			fmt.Println(msg)
			return nil
		},
	}

	// Support CLI flags for configuration (config file, context, overrides, output, debug)
	// Config file path (default: .valet.yaml)
	cmd.PersistentFlags().String("config-file", ".valet.yaml", "config file path (default: .valet.yaml)")
	cmd.PersistentFlags().StringP("context", "c", ".", "context directory containing values.yaml (optional)")
	cmd.PersistentFlags().StringP("overrides", "f", "", "overrides file (optional)")
	cmd.PersistentFlags().StringP("output", "o", "values.schema.json", "output file (default: values.schema.json)")
	cmd.PersistentFlags().BoolP("debug", "d", false, "enable debug logging")

	// add subcommands
	cmd.AddCommand(NewVersionCmd())
	cmd.AddCommand(NewGenerateCmd())

	return cmd
}

// initializeConfig loads configuration from file and applies CLI flags
func initializeConfig(cmd *cobra.Command) (*config.Config, error) {
	// Read config file
	cfgFile, _ := cmd.Flags().GetString("config-file")
	c, err := config.LoadConfig(cfgFile)
	if err != nil {
		return nil, err
	}
	// Override with CLI flags or defaults
	// Context: default to value or override
	cliCtx, _ := cmd.Flags().GetString("context")
	if cmd.Flags().Changed("context") || c.Context == "" {
		c.Context = cliCtx
	}
	if cmd.Flags().Changed("overrides") {
		ov, _ := cmd.Flags().GetString("overrides")
		c.Overrides = ov
	}
	if cmd.Flags().Changed("output") {
		out, _ := cmd.Flags().GetString("output")
		c.Output = out
	}
	if cmd.Flags().Changed("debug") {
		dbg, _ := cmd.Flags().GetBool("debug")
		c.Debug = dbg
	}
	if c.Debug {
		log.Printf("Config: %+v\n", c)
	}
	return c, nil
}

// (bindFlags removed; flags now override config file values directly)
