package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mkm29/valet/internal/config"
	"github.com/mkm29/valet/internal/telemetry"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	cfg *config.Config
	tel *telemetry.Telemetry
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "valet",
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

			// Initialize logger based on debug setting
			var logConfig zap.Config
			if cfg.Debug {
				logConfig = zap.NewDevelopmentConfig()
				logConfig.EncoderConfig.TimeKey = "timestamp"
			} else {
				logConfig = zap.NewProductionConfig()
			}

			logger, err := logConfig.Build()
			if err != nil {
				return fmt.Errorf("failed to initialize logger: %w", err)
			}
			zap.ReplaceGlobals(logger)

			// Log config if debug is enabled
			if cfg.Debug {
				zap.L().Debug("Config loaded", zap.Any("config", cfg))
			}

			// Initialize telemetry if enabled
			if tel == nil && cfg.Telemetry != nil && cfg.Telemetry.Enabled {
				ctx := cmd.Context()
				t, err := telemetry.Initialize(ctx, cfg.Telemetry)
				if err != nil {
					// Log error but don't fail - telemetry is optional
					zap.L().Debug("Failed to initialize telemetry", zap.Error(err))
				} else {
					tel = t
				}
			}

			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			// Shutdown telemetry if it was initialized
			if tel != nil {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				if err := tel.Shutdown(shutdownCtx); err != nil {
					zap.L().Error("Error shutting down telemetry", zap.Error(err))
					// Don't return error - telemetry shutdown failure shouldn't fail the command
				}
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

	// Telemetry flags
	cmd.PersistentFlags().Bool("telemetry-enabled", false, "enable telemetry")
	cmd.PersistentFlags().String("telemetry-exporter", "none", "telemetry exporter type (none, stdout, otlp)")
	cmd.PersistentFlags().String("telemetry-endpoint", "localhost:4317", "OTLP endpoint for telemetry")
	cmd.PersistentFlags().Bool("telemetry-insecure", false, "use insecure connection for OTLP")
	cmd.PersistentFlags().Float64("telemetry-sample-rate", 1.0, "trace sampling rate (0.0 to 1.0)")

	// add subcommands
	cmd.AddCommand(NewVersionCmd())
	cmd.AddCommand(NewGenerateCmd())

	return cmd
}

// initializeConfig loads configuration from file and applies CLI flags
func initializeConfig(cmd *cobra.Command) (*config.Config, error) {
	// Load config file if specified
	var c *config.Config
	var err error
	// Get the root command to access persistent flags
	rootCmd := cmd.Root()
	cfgFile, _ := rootCmd.PersistentFlags().GetString("config-file")

	// Check if config file exists (either explicitly set or default)
	if _, statErr := os.Stat(cfgFile); statErr == nil {
		c, err = config.LoadConfig(cfgFile)
		if err != nil {
			return nil, err
		}
	} else if rootCmd.PersistentFlags().Changed("config-file") {
		// Config file was explicitly specified but doesn't exist
		return nil, fmt.Errorf("config file not found: %s", cfgFile)
	} else {
		// No config file: start with empty config
		c = &config.Config{
			Telemetry: config.NewTelemetryConfig(),
		}
	}

	// Always set the service version from build info, regardless of config source
	if c.Telemetry != nil {
		c.Telemetry.ServiceVersion = GetBuildVersion()
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

	// Handle telemetry flags
	if c.Telemetry == nil {
		c.Telemetry = config.NewTelemetryConfig()
	}

	if cmd.PersistentFlags().Changed("telemetry-enabled") {
		enabled, _ := cmd.PersistentFlags().GetBool("telemetry-enabled")
		c.Telemetry.Enabled = enabled
	}
	if cmd.PersistentFlags().Changed("telemetry-exporter") {
		exporter, _ := cmd.PersistentFlags().GetString("telemetry-exporter")
		c.Telemetry.ExporterType = exporter
	}
	if cmd.PersistentFlags().Changed("telemetry-endpoint") {
		endpoint, _ := cmd.PersistentFlags().GetString("telemetry-endpoint")
		c.Telemetry.OTLPEndpoint = endpoint
	}
	if cmd.PersistentFlags().Changed("telemetry-insecure") {
		insecure, _ := cmd.PersistentFlags().GetBool("telemetry-insecure")
		c.Telemetry.Insecure = insecure
	}
	if cmd.PersistentFlags().Changed("telemetry-sample-rate") {
		rate, _ := cmd.PersistentFlags().GetFloat64("telemetry-sample-rate")
		c.Telemetry.SampleRate = rate
	}

	return c, nil
}

// GetTelemetry returns the global telemetry instance
func GetTelemetry() *telemetry.Telemetry {
	return tel
}

// (bindFlags removed; flags now override config file values directly)
