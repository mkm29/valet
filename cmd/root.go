package cmd

import (
	"fmt"
	"log"

	"github.com/mkm29/schemagen/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	v   *viper.Viper
	cfg *config.Config
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schemagen",
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
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("running schemagen")
		},
	}

	cmd.PersistentFlags().StringP("overrides", "f", "", "overrides file (optional)")
	cmd.PersistentFlags().StringP("output", "o", "values.schema.json", "output file (default: values.schema.json)")

	v := viper.New()

	v.BindPFlag("overrides", cmd.PersistentFlags().Lookup("overrides"))
	v.BindPFlag("output", cmd.PersistentFlags().Lookup("output"))

	// add subcommands
	cmd.AddCommand(NewVersionCmd())
	cmd.AddCommand(NewGenerateCmd())

	return cmd
}
func initializeConfig(cmd *cobra.Command) (*config.Config, error) {
	// Initialize config
	v = viper.New()
	c, err := config.LoadConfig(v)
	if err != nil {
		return nil, err
	}
	cfg = c
	if c.Debug {
		log.Printf("Config: %+v\n", cfg)
	}

	bindFlags(cmd, v)

	return c, nil
}

// Bind each cobra flag to its associated viper configuration (config file and environment variable)
func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Determine the naming convention of the flags when represented in the config file
		configName := f.Name

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && v.IsSet(configName) {
			val := v.Get(configName)
			if err := cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val)); err != nil {
				log.Fatalf("unable to set flag '%s' from config: %v", f.Name, err)
			}
		}
	})
}
