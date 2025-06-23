package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/mkm29/valet/internal/config"
	"github.com/mkm29/valet/internal/telemetry"
	"github.com/mkm29/valet/internal/utils"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// CommandContext extends cobra.Command to carry dependencies
type CommandContext struct {
	*cobra.Command
	App *App
}

// NewRootCmd creates a new root command with a fresh App instance
func NewRootCmd() *cobra.Command {
	return NewRootCmdWithApp(NewApp())
}

// NewRootCmdWithApp creates a root command with dependency injection
func NewRootCmdWithApp(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "valet",
		Short: "JSON Schema Generator",
		Long:  `A JSON Schema Generator for Helm charts and other YAML files.`,
		// Do not print usage on error; just show the error message
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Initialize configuration if not already set
			if app.Config == nil {
				c, err := initializeConfig(cmd)
				if err != nil {
					return err
				}
				app.Config = c
			}

			// Initialize logger based on log level
			cleanup, err := app.InitializeLogger(app.Config.LogLevel.Level)
			if err != nil {
				return fmt.Errorf("failed to initialize logger: %w", err)
			}
			// Store the cleanup function in the app for later use
			app.loggerCleanup = cleanup

			// Log config if debug level is enabled
			if app.Config.LogLevel.Level == zapcore.DebugLevel {
				logDebugConfiguration(app.Logger, app.Config)
			}

			// Initialize telemetry if enabled
			if app.Telemetry == nil && app.Config.Telemetry != nil && app.Config.Telemetry.Enabled {
				ctx := cmd.Context()
				t, err := telemetry.Initialize(ctx, app.Config.Telemetry)
				if err != nil {
					// Log error but don't fail - telemetry is optional
					app.Logger.Debug("Failed to initialize telemetry", zap.Error(err))
				} else {
					app.Telemetry = t
				}
			}

			// Store app in command context for subcommands
			cmd.SetContext(context.WithValue(cmd.Context(), appContextKey, app))

			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			// Shutdown telemetry if it was initialized
			if app.Telemetry != nil {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				if err := app.Telemetry.Shutdown(shutdownCtx); err != nil {
					if app.Logger != nil {
						app.Logger.Error("Error shutting down telemetry", zap.Error(err))
					}
				}
			}

			// Flush logger if it was initialized
			if app.loggerCleanup != nil {
				app.loggerCleanup()
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default action: delegate to Generate
			ctx := app.Config.Context
			if len(args) > 0 && args[0] != "" {
				ctx = args[0]
			}
			if ctx == "" {
				return cmd.Help()
			}
			msg, err := GenerateWithApp(app, ctx, app.Config.Overrides)
			if err != nil {
				return err
			}
			fmt.Println(msg)
			return nil
		},
	}

	// Add persistent flags
	addPersistentFlags(cmd)

	// Add subcommands with app context
	cmd.AddCommand(NewVersionCmdWithApp(app))
	cmd.AddCommand(NewGenerateCmdWithApp(app))

	return cmd
}

// logDebugConfiguration logs the configuration in debug mode
func logDebugConfiguration(logger *zap.Logger, cfg *config.Config) {
	// Pretty print configuration to stdout as JSON
	configJSON, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		logger.Error("Failed to marshal config", zap.Error(err))
	} else {
		fmt.Println("=== Valet Configuration ===")
		fmt.Println(string(configJSON))
		fmt.Println("===========================")
	}

	// Also log with structured fields for debugging
	fields := buildConfigFields(cfg)
	logger.Debug("Configuration loaded", fields...)
}

// buildConfigFields builds zap fields from config
func buildConfigFields(cfg *config.Config) []zap.Field {
	fields := []zap.Field{
		zap.String("logLevel", cfg.LogLevel.String()),
		zap.String("context", cfg.Context),
		zap.String("overrides", cfg.Overrides),
		zap.String("output", cfg.Output),
	}

	// Add telemetry config if present
	if cfg.Telemetry != nil {
		fields = append(fields,
			zap.Bool("telemetry.enabled", cfg.Telemetry.Enabled),
			zap.String("telemetry.serviceName", cfg.Telemetry.ServiceName),
			zap.String("telemetry.serviceVersion", cfg.Telemetry.ServiceVersion),
			zap.String("telemetry.exporterType", cfg.Telemetry.ExporterType),
			zap.String("telemetry.otlpEndpoint", cfg.Telemetry.OTLPEndpoint),
			zap.Bool("telemetry.insecure", cfg.Telemetry.Insecure),
			zap.Float64("telemetry.sampleRate", cfg.Telemetry.SampleRate),
		)
	}

	// Add helm config if present
	if cfg.Helm != nil && cfg.Helm.Chart != nil {
		fields = append(fields,
			zap.String("helm.chart.name", cfg.Helm.Chart.Name),
			zap.String("helm.chart.version", cfg.Helm.Chart.Version),
		)
		if cfg.Helm.Chart.Registry != nil {
			fields = append(fields,
				zap.String("helm.chart.registry.url", cfg.Helm.Chart.Registry.URL),
				zap.String("helm.chart.registry.type", cfg.Helm.Chart.Registry.Type),
				zap.Bool("helm.chart.registry.insecure", cfg.Helm.Chart.Registry.Insecure),
			)
			if cfg.Helm.Chart.Registry.Auth != nil {
				auth := cfg.Helm.Chart.Registry.Auth
				maskedUsername := auth.Username
				maskedPassword := "[REDACTED]"
				if auth.Password != "" {
					maskedPassword = "[REDACTED]"
				}
				fields = append(fields,
					zap.String("helm.registry.auth.username", maskedUsername),
					zap.String("helm.registry.auth.password", maskedPassword),
				)
			}
		}
	}

	return fields
}

// addPersistentFlags adds all persistent flags to the command
func addPersistentFlags(cmd *cobra.Command) {
	// Config file path (default: .valet.yaml)
	cmd.PersistentFlags().String("config-file", ".valet.yaml", "config file path (default: .valet.yaml)")
	cmd.PersistentFlags().StringP("context", "c", ".", "context directory containing values.yaml (optional)")
	cmd.PersistentFlags().StringP("overrides", "f", "", "overrides file (optional)")
	cmd.PersistentFlags().StringP("output", "o", "values.schema.json", "output file (default: values.schema.json)")
	cmd.PersistentFlags().StringP("log-level", "l", "info", "log level (debug, info, warn, error, dpanic, panic, fatal)")

	// Telemetry flags
	cmd.PersistentFlags().Bool("telemetry-enabled", false, "enable telemetry")
	cmd.PersistentFlags().String("telemetry-exporter", "none", "telemetry exporter type (none, stdout, otlp)")
	cmd.PersistentFlags().String("telemetry-endpoint", "localhost:4317", "OTLP endpoint for telemetry")
	cmd.PersistentFlags().Bool("telemetry-insecure", false, "use insecure connection for OTLP")
	cmd.PersistentFlags().Float64("telemetry-sample-rate", 1.0, "trace sampling rate (0.0 to 1.0)")
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
		// No config file: start with default config
		c = config.NewConfig()
	}

	// Always set the service version from build info, regardless of config source
	if c.Telemetry != nil {
		c.Telemetry.ServiceVersion = utils.GetBuildVersion()
	}

	// Apply CLI flag overrides
	applyFlagOverrides(cmd, c)

	return c, nil
}

// applyFlagOverrides applies CLI flag values to config
func applyFlagOverrides(cmd *cobra.Command, c *config.Config) {
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
	if cmd.PersistentFlags().Changed("log-level") {
		levelStr, _ := cmd.PersistentFlags().GetString("log-level")
		// Parse the log level string to zapcore.Level
		var level zapcore.Level
		if err := level.UnmarshalText([]byte(levelStr)); err != nil {
			// Default to info level if parsing fails
			level = zapcore.InfoLevel
		}
		c.LogLevel = config.LogLevel{Level: level}
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
}

// GetAppFromContext extracts App from command context
func GetAppFromContext(cmd *cobra.Command) *App {
	if app, ok := cmd.Context().Value(appContextKey).(*App); ok {
		return app
	}
	// Fallback to new app if not found (for backward compatibility)
	return NewApp()
}
