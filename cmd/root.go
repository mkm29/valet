package cmd

import (
	"fmt"
	"log"

	"github.com/mkm29/valet/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	v   *viper.Viper
	cfg *config.Config
)

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
	cmd.PersistentFlags().String("config-file", ".valet.yaml", "config file path (default is .valet.yaml)")
	cmd.PersistentFlags().StringP("context", "c", ".", "context directory containing values.yaml (optional)")
	cmd.PersistentFlags().StringP("overrides", "f", "", "overrides file (optional)")
	cmd.PersistentFlags().StringP("output", "o", "values.schema.json", "output file (default: values.schema.json)")
	cmd.PersistentFlags().BoolP("debug", "d", false, "enable debug logging")

	// add subcommands for explicit operations
	cmd.AddCommand(NewVersionCmd())
	cmd.AddCommand(NewGenerateCmd())

	return cmd
}

// initializeConfig loads configuration from file/environment and applies CLI flags
func initializeConfig(cmd *cobra.Command) (*config.Config, error) {
	// Setup viper and read config file (if exists) via config-file flag
	v = viper.New()
	cfgFile, _ := cmd.Flags().GetString("config-file")
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	}
	// Load config (reads file and environment variables)
	c, err := config.LoadConfig(v)
	if err != nil {
		return nil, err
	}
	// Override config with CLI flags if set
	if cmd.Flags().Changed("context") {
		ctx, _ := cmd.Flags().GetString("context")
		c.Context = ctx
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
