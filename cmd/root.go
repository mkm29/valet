package cmd

import (
   "fmt"
   "log"
   "os"
   "path/filepath"

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
           // Ensure a values file exists
           pathYml := filepath.Join(ctx, "values.yaml")
           if _, err := os.Stat(pathYml); err != nil {
               // try .yml
               pathYml2 := filepath.Join(ctx, "values.yml")
               if _, err2 := os.Stat(pathYml2); err2 != nil {
                   return fmt.Errorf("no values.yaml or values.yml found in %s", ctx)
               }
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
