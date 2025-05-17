package cmd

import (
	"fmt"
	"log"

	"github.com/mkm29/schemagen/internal/config"
	"github.com/spf13/cobra"
)

var cfg *config.Config

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schemagen",
		Short: "JSON Schema Generator",
		Long:  `A JSON Schema Generator for Helm charts and other YAML files.`,
		// Do not print usage on error; just show the error message
		SilenceUsage: true,
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
			// Default action: delegate to Generate
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
	// Only read config file if flag explicitly set
	var c *config.Config
	var err error
	if cmd.PersistentFlags().Changed("config-file") {
		cfgFile, _ := cmd.PersistentFlags().GetString("config-file")
		c, err = config.LoadConfig(cfgFile)
		if err != nil {
			return nil, err
		}
	} else {
		// No config file: start with empty config
		c = &config.Config{}
	}
	// Override with CLI flags or defaults
	// Context: default to value or override
	// Context flag override
	cliCtx, _ := cmd.PersistentFlags().GetString("context")
	if cmd.PersistentFlags().Changed("context") || c.Context == "" {
		c.Context = cliCtx
	}
	if cmd.PersistentFlags().Changed("overrides") {
		ov, _ := cmd.PersistentFlags().GetString("overrides")
		c.Overrides = ov
	}
	if cmd.PersistentFlags().Changed("output") {
		out, _ := cmd.PersistentFlags().GetString("output")
		c.Output = out
	}
	if cmd.PersistentFlags().Changed("debug") {
		dbg, _ := cmd.PersistentFlags().GetBool("debug")
		c.Debug = dbg
	}
	if c.Debug {
		log.Printf("Config: %+v\n", c)
	}
	return c, nil
}

// (bindFlags removed; flags now override config file values directly)
