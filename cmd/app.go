package cmd

import (
	"log/slog"
	"os"

	"github.com/mkm29/valet/internal/config"
	"github.com/mkm29/valet/internal/telemetry"
)

// contextKey is a type for context keys to avoid collisions
type contextKey string

// appContextKey is the context key for storing the App instance
const appContextKey contextKey = "app"

// App holds the application dependencies
type App struct {
	Config        *config.Config
	Telemetry     *telemetry.Telemetry
	Logger        *slog.Logger
	loggerCleanup func()
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
func (a *App) WithLogger(logger *slog.Logger) *App {
	a.Logger = logger
	return a
}

// InitializeLogger creates a new logger based on log level
// Returns a cleanup function that should be deferred to ensure logs are flushed
func (a *App) InitializeLogger(level slog.Level) (func(), error) {
	logger, err := createLogger(level)
	if err != nil {
		return nil, err
	}
	a.Logger = logger
	slog.SetDefault(logger)

	// Return a cleanup function (no-op for slog)
	cleanup := func() {
		// slog doesn't require explicit syncing
	}

	return cleanup, nil
}

// createLogger creates a new slog logger based on log level
func createLogger(level slog.Level) (*slog.Logger, error) {
	// Create handler options with the specified level
	opts := &slog.HandlerOptions{
		Level: level,
		AddSource: level == slog.LevelDebug, // Add source info in debug mode
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize attribute names for consistency
			switch a.Key {
			case slog.TimeKey:
				return slog.Attr{Key: "timestamp", Value: a.Value}
			}
			return a
		},
	}

	// Use text handler for debug level (more readable), JSON for others
	var handler slog.Handler
	if level == slog.LevelDebug {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler), nil
}
