package cmd

import (
	"github.com/mkm29/valet/internal/config"
	"github.com/mkm29/valet/internal/telemetry"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// contextKey is a type for context keys to avoid collisions
type contextKey string

// appContextKey is the context key for storing the App instance
const appContextKey contextKey = "app"

// App holds the application dependencies
type App struct {
	Config    *config.Config
	Telemetry *telemetry.Telemetry
	Logger    *zap.Logger
}

// NewApp creates a new App instance
func NewApp() *App {
	return &App{}
}

// WithConfig sets the config
func (a *App) WithConfig(cfg *config.Config) *App {
	a.Config = cfg
	return a
}

// WithTelemetry sets the telemetry
func (a *App) WithTelemetry(tel *telemetry.Telemetry) *App {
	a.Telemetry = tel
	return a
}

// WithLogger sets the logger
func (a *App) WithLogger(logger *zap.Logger) *App {
	a.Logger = logger
	return a
}

// InitializeLogger creates a new logger based on debug setting
func (a *App) InitializeLogger(debug bool) error {
	logger, err := createLogger(debug)
	if err != nil {
		return err
	}
	a.Logger = logger
	zap.ReplaceGlobals(logger)
	return nil
}

// createLogger creates a new zap logger based on debug setting
func createLogger(debug bool) (*zap.Logger, error) {
	var logConfig zap.Config
	if debug {
		logConfig = zap.NewDevelopmentConfig()
		logConfig.EncoderConfig.TimeKey = "timestamp"
		logConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		// Use console encoder for more readable output
		logConfig.Encoding = "console"
		logConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		logConfig = zap.NewProductionConfig()
		logConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	return logConfig.Build()
}
